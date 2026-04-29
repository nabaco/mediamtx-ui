package api

import (
	"database/sql"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"mediamtx-ui/internal/auth"
	"mediamtx-ui/internal/config"
	"mediamtx-ui/internal/mediamtx"
	"mediamtx-ui/internal/parser"
	"mediamtx-ui/internal/sysdetect"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	cfg        *config.Config
	db         *sql.DB
	jwt        *auth.Manager
	mtx        *mediamtx.Client
	configFile *parser.ParseResult
	deployType sysdetect.DeployType
}

func NewServer(
	cfg *config.Config,
	db *sql.DB,
	jwt *auth.Manager,
	mtx *mediamtx.Client,
	configFile *parser.ParseResult,
	deployType sysdetect.DeployType,
) *Server {
	return &Server{
		cfg:        cfg,
		db:         db,
		jwt:        jwt,
		mtx:        mtx,
		configFile: configFile,
		deployType: deployType,
	}
}

// Handler builds and returns the complete chi router.
func (s *Server) Handler(frontendFS fs.FS) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// Prometheus metrics endpoint (no auth — restrict via network/firewall)
	r.Handle("/metrics", promhttp.Handler())

	// mediamtx auth callback (called by mediamtx, no UI JWT required)
	r.Post("/api/v1/mediamtx/auth", s.handleMediaMTXAuth)

	// Public routes
	r.Post("/api/v1/auth/login", s.handleLogin)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)

		r.Get("/api/v1/auth/me", s.handleMe)
		r.Post("/api/v1/auth/change-password", s.handleChangePassword)
		r.Get("/api/v1/auth/stream-token", s.handleGetStreamToken)
		r.Post("/api/v1/auth/stream-token", s.handleRegenerateOwnStreamToken)

		// Streams — visible based on user ACL
		r.Get("/api/v1/streams", s.handleListStreams)
		r.Get("/api/v1/streams/{name}", s.handleGetStream)
		r.Get("/api/v1/streams/{name}/urls", s.handleStreamURLs)

		// System info
		r.Get("/api/v1/system/info", s.handleSystemInfo)

		// Admin-only
		r.Group(func(r chi.Router) {
			r.Use(s.requireAdmin)

			// Stream management
			r.Post("/api/v1/streams", s.handleCreateStream)
			r.Get("/api/v1/streams/{name}/config", s.handleGetStreamConfig)
			r.Patch("/api/v1/streams/{name}", s.handleUpdateStream)
			r.Delete("/api/v1/streams/{name}", s.handleDeleteStream)

			// Users
			r.Get("/api/v1/users", s.handleListUsers)
			r.Post("/api/v1/users", s.handleCreateUser)
			r.Get("/api/v1/users/{id}", s.handleGetUser)
			r.Patch("/api/v1/users/{id}", s.handleUpdateUser)
			r.Delete("/api/v1/users/{id}", s.handleDeleteUser)
			r.Post("/api/v1/users/{id}/stream-token", s.handleRegenerateStreamToken)

			// Groups
			r.Get("/api/v1/groups", s.handleListGroups)
			r.Post("/api/v1/groups", s.handleCreateGroup)
			r.Patch("/api/v1/groups/{id}", s.handleRenameGroup)
			r.Delete("/api/v1/groups/{id}", s.handleDeleteGroup)
			r.Get("/api/v1/groups/{id}/members", s.handleListGroupMembers)
			r.Post("/api/v1/groups/{id}/members", s.handleAddGroupMember)
			r.Delete("/api/v1/groups/{id}/members/{userId}", s.handleRemoveGroupMember)

			// ACLs
			r.Get("/api/v1/acls", s.handleListACLs)
			r.Post("/api/v1/acls", s.handleCreateACL)
			r.Delete("/api/v1/acls/{id}", s.handleDeleteACL)

			// Config (mediamtx config file display)
			r.Get("/api/v1/system/config", s.handleSystemConfig)

			// Audit log
			r.Get("/api/v1/audit", s.handleListAuditLog)
		})
	})

	// Frontend: dev proxy or embedded static files
	if devProxy := os.Getenv("MEDIAMTX_UI_DEV_PROXY"); devProxy != "" {
		target, _ := url.Parse(devProxy)
		proxy := httputil.NewSingleHostReverseProxy(target)
		r.Handle("/*", proxy)
	} else {
		r.Handle("/*", spaHandler(frontendFS))
	}

	return r
}

// spaHandler serves a static filesystem and falls back to index.html for SPA routes.
// The FS is expected to have a top-level "dist" directory (from internal/web/embed.go).
func spaHandler(fsys fs.FS) http.Handler {
	sub, err := fs.Sub(fsys, "dist")
	if err != nil {
		sub = fsys
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If path has a file extension or is explicitly a file, try to serve it
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}

		if _, err := fs.Stat(sub, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fall back to index.html for all other paths (SPA hash routing)
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
