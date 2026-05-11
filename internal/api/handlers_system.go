package api

import (
	"net/http"

	"mediamtx-ui/internal/parser"
)

type systemInfoResponse struct {
	Version        string `json:"version"`
	DeployType     string `json:"deployType"`
	MediaMTXHost   string `json:"mediamtxHost"`
	RTSPPort       int    `json:"rtspPort"`
	HLSPort        int    `json:"hlsPort"`
	WebRTCPort     int    `json:"webrtcPort"`
	RTMPPort       int    `json:"rtmpPort"`
	MediaMTXOnline bool   `json:"mediamtxOnline"`
}

type systemConfigResponse struct {
	Available    bool   `json:"available"`
	ResolvedPath string `json:"resolvedPath,omitempty"`
	RawYAML      string `json:"rawYaml,omitempty"`
}

// Version is set at build time via -ldflags.
var Version = "dev"

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	online := s.mtx.Ping() == nil

	jsonOK(w, systemInfoResponse{
		Version:        Version,
		DeployType:     string(s.deployType),
		MediaMTXHost:   s.mediamtxPublicHostFor(r),
		RTSPPort:       s.cfg.MediaMTX.RTSPPort,
		HLSPort:        s.cfg.MediaMTX.HLSPort,
		WebRTCPort:     s.cfg.MediaMTX.WebRTCPort,
		RTMPPort:       s.cfg.MediaMTX.RTMPPort,
		MediaMTXOnline: online,
	})
}

func (s *Server) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	// Re-read on every request so edits (including ones we wrote via
	// parser.UpsertPath) show up without a server restart. Falls back to
	// the path the parser resolved at startup when no explicit ConfigPath
	// is set in config.
	path := s.cfg.MediaMTX.ConfigPath
	if path == "" && s.configFile != nil {
		path = s.configFile.ResolvedPath
	}
	cur := parser.Parse(path)
	if cur == nil || !cur.Available {
		jsonOK(w, systemConfigResponse{Available: false})
		return
	}
	jsonOK(w, systemConfigResponse{
		Available:    true,
		ResolvedPath: cur.ResolvedPath,
		RawYAML:      cur.RawYAML,
	})
}

func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	auditLogHandler(s)(w, r)
}
