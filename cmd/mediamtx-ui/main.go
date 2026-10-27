package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"mediamtx-ui/internal/api"
	"mediamtx-ui/internal/auth"
	"mediamtx-ui/internal/config"
	"mediamtx-ui/internal/db"
	"mediamtx-ui/internal/mediamtx"
	"mediamtx-ui/internal/parser"
	"mediamtx-ui/internal/sysdetect"
	"mediamtx-ui/internal/web"
)

// Version is set at build time: -ldflags="-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	setupLogger(cfg.Log.Level)
	slog.Info("mediamtx-ui starting", "version", Version)

	database, err := db.Open(cfg.DB.Path)
	if err != nil {
		slog.Error("database open failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()
	slog.Info("database opened", "path", cfg.DB.Path)

	startAuditMaintenance(database, cfg.Audit.RetentionDays)

	if err := seedAdmin(database, cfg); err != nil {
		slog.Error("admin seed failed", "err", err)
		os.Exit(1)
	}

	mtxClient := mediamtx.NewClient(cfg.MediaMTX.APIAddress, cfg.MediaMTX.APIUser, cfg.MediaMTX.APIPass, cfg.MediaMTX.APIKey)
	if err := mtxClient.Ping(); err != nil {
		slog.Warn("mediamtx not reachable at startup", "addr", cfg.MediaMTX.APIAddress, "err", err)
	} else {
		slog.Info("mediamtx connected", "addr", cfg.MediaMTX.APIAddress)
	}

	configFile := parser.Parse(cfg.MediaMTX.ConfigPath)
	if configFile.Available {
		slog.Info("mediamtx config file found", "path", configFile.ResolvedPath)
	} else {
		slog.Info("mediamtx config file not available (remote mode)")
	}

	deployType := sysdetect.Detect()
	slog.Info("deployment type detected", "type", deployType)

	jwtMgr := auth.NewManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry)

	api.Version = Version
	srv := api.NewServer(cfg, database, jwtMgr, mtxClient, configFile, deployType)
	handler := srv.Handler(web.FS)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("HTTP server listening", "addr", addr)
	if err := httpSrv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// startAuditMaintenance prunes expired audit rows now and once a day after
// that. It runs in the background so a first prune over a long-neglected table
// doesn't delay the listener coming up.
func startAuditMaintenance(database *sql.DB, retentionDays int) {
	if retentionDays <= 0 {
		slog.Warn("audit log pruning disabled, table will grow unbounded")
		return
	}

	prune := func() {
		start := time.Now()
		n, err := db.PruneAuditLog(database, retentionDays)
		if err != nil {
			slog.Error("audit log prune failed", "err", err)
			return
		}
		if n == 0 {
			return
		}
		slog.Info("audit log pruned", "rows", n, "retentionDays", retentionDays, "took", time.Since(start))

		// Only reclaim the file after a big purge; routine daily deletes leave
		// free pages that SQLite reuses, and VACUUM rewrites the whole DB.
		if n >= 100_000 {
			if err := db.VacuumDB(database); err != nil {
				slog.Error("audit log vacuum failed", "err", err)
			} else {
				slog.Info("database vacuumed", "took", time.Since(start))
			}
		}
	}

	go func() {
		prune()
		for range time.Tick(24 * time.Hour) {
			prune()
		}
	}()
}

// seedAdmin creates the initial admin account if the users table is empty.
func seedAdmin(database *sql.DB, cfg *config.Config) error {
	n, err := db.CountUsers(database)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Auth.InitialAdminPass), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	if _, err := db.CreateUser(database, cfg.Auth.InitialAdminUser, string(hash), db.RoleAdmin); err != nil {
		return fmt.Errorf("create initial admin: %w", err)
	}

	slog.Info("initial admin account created", "username", cfg.Auth.InitialAdminUser)
	return nil
}

func setupLogger(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
