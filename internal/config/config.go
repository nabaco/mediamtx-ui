package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	MediaMTX MediaMTXConfig
	Auth     AuthConfig
	DB       DBConfig
	Log      LogConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type MediaMTXConfig struct {
	// APIAddress is the base URL for mediamtx's management API (e.g. http://localhost:9997)
	APIAddress string
	// APIUser / APIPass are the HTTP Basic Auth credentials for the mediamtx API.
	// Set these when mediamtx has auth enabled (api: yes with internal users).
	APIUser string
	APIPass string
	// APIKey is an optional Bearer token (alternative to user/pass, rarely used)
	APIKey string
	// RTSPPort, HLSPort, WebRTCPort, RTMPPort are the public-facing stream ports.
	// These are used to generate stream URLs for users.
	RTSPPort   int
	HLSPort    int
	WebRTCPort int
	RTMPPort   int
	SRTPort    int
	// PublicHost is the hostname/IP that end users can reach mediamtx on.
	// Defaults to the host part of APIAddress.
	PublicHost string
	// ConfigPath is the optional path to mediamtx.yml for config parsing.
	// Searched in standard locations if unset.
	ConfigPath string
}

type AuthConfig struct {
	// JWTSecret is the HMAC-SHA256 signing secret for UI JWTs.
	// Must be set in production.
	JWTSecret string
	// JWTExpiry is how long UI JWTs remain valid.
	JWTExpiry time.Duration
	// InitialAdminUser / InitialAdminPass are used once to seed the admin account
	// if no users exist in the database.
	InitialAdminUser string
	InitialAdminPass string
}

type DBConfig struct {
	Path string
}

type LogConfig struct {
	Level string // debug, info, warn, error
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("MEDIAMTX_UI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Server defaults
	v.SetDefault("server.host", "")
	v.SetDefault("server.port", 9996)

	// mediamtx defaults
	v.SetDefault("mediamtx.api_address", "http://localhost:9997")
	v.SetDefault("mediamtx.api_user", "")
	v.SetDefault("mediamtx.api_pass", "")
	v.SetDefault("mediamtx.api_key", "")
	v.SetDefault("mediamtx.rtsp_port", 8554)
	v.SetDefault("mediamtx.hls_port", 8888)
	v.SetDefault("mediamtx.webrtc_port", 8889)
	v.SetDefault("mediamtx.rtmp_port", 1935)
	v.SetDefault("mediamtx.srt_port", 8890)
	v.SetDefault("mediamtx.public_host", "")
	v.SetDefault("mediamtx.config_path", "")

	// Auth defaults
	v.SetDefault("auth.jwt_secret", "change-me-in-production")
	v.SetDefault("auth.jwt_expiry", "24h")
	v.SetDefault("auth.initial_admin_user", "admin")
	v.SetDefault("auth.initial_admin_pass", "admin")

	// DB defaults
	v.SetDefault("db.path", "mediamtx-ui.db")

	// Log defaults
	v.SetDefault("log.level", "info")

	// Optionally load config file
	v.SetConfigName("mediamtx-ui")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/mediamtx-ui/")
	_ = v.ReadInConfig() // ignore missing file

	expiry, err := time.ParseDuration(v.GetString("auth.jwt_expiry"))
	if err != nil {
		return nil, fmt.Errorf("invalid auth.jwt_expiry: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: v.GetString("server.host"),
			Port: v.GetInt("server.port"),
		},
		MediaMTX: MediaMTXConfig{
			APIAddress: v.GetString("mediamtx.api_address"),
			APIUser:    v.GetString("mediamtx.api_user"),
			APIPass:    v.GetString("mediamtx.api_pass"),
			APIKey:     v.GetString("mediamtx.api_key"),
			RTSPPort:   v.GetInt("mediamtx.rtsp_port"),
			HLSPort:    v.GetInt("mediamtx.hls_port"),
			WebRTCPort: v.GetInt("mediamtx.webrtc_port"),
			RTMPPort:   v.GetInt("mediamtx.rtmp_port"),
			SRTPort:    v.GetInt("mediamtx.srt_port"),
			PublicHost: v.GetString("mediamtx.public_host"),
			ConfigPath: v.GetString("mediamtx.config_path"),
		},
		Auth: AuthConfig{
			JWTSecret:        v.GetString("auth.jwt_secret"),
			JWTExpiry:        expiry,
			InitialAdminUser: v.GetString("auth.initial_admin_user"),
			InitialAdminPass: v.GetString("auth.initial_admin_pass"),
		},
		DB: DBConfig{
			Path: v.GetString("db.path"),
		},
		Log: LogConfig{
			Level: v.GetString("log.level"),
		},
	}

	return cfg, nil
}
