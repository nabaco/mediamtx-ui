package api

import (
	"net/http"
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
	if s.configFile == nil || !s.configFile.Available {
		jsonOK(w, systemConfigResponse{Available: false})
		return
	}
	jsonOK(w, systemConfigResponse{
		Available:    true,
		ResolvedPath: s.configFile.ResolvedPath,
		RawYAML:      s.configFile.RawYAML,
	})
}

func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	auditLogHandler(s)(w, r)
}
