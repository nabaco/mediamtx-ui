package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	dbpkg "mediamtx-ui/internal/db"
	"mediamtx-ui/internal/mediamtx"
	"mediamtx-ui/internal/metrics"
)

type streamResponse struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Source        string   `json:"source,omitempty"`
	Ready         bool     `json:"ready"`
	Tracks        []string `json:"tracks"`
	Readers       int      `json:"readers"`
	BytesReceived uint64   `json:"bytesReceived"`
	BytesSent     uint64   `json:"bytesSent"`
}

type streamURLsResponse struct {
	RTSP            string `json:"rtsp"`
	RTSPS           string `json:"rtsps,omitempty"`
	HLS             string `json:"hls"`
	WebRTC          string `json:"webrtc"`
	RTMP            string `json:"rtmp"`
	SRT             string `json:"srt,omitempty"`
	StreamToken     string `json:"streamToken,omitempty"`
	Username        string `json:"username,omitempty"`
	IsPublishStream bool   `json:"isPublishStream"`
}

type createStreamRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Source         string `json:"source"`
	SourceOnDemand bool   `json:"sourceOnDemand"`
	Record         bool   `json:"record"`
	MaxReaders     int    `json:"maxReaders"`
}

func (s *Server) handleListStreams(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)

	paths, err := s.mtx.ListAllPaths()
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "mediamtx unreachable: "+err.Error())
		return
	}

	metaMap, _ := dbpkg.ListStreamMeta(s.db)

	var out []streamResponse
	for _, p := range paths {
		// Admins see all; regular users only see streams they can read
		if claims.Role != string(dbpkg.RoleAdmin) {
			ok, _ := dbpkg.CheckAccess(s.db, claims.UserID, p.Name, dbpkg.ACLActionRead)
			if !ok {
				continue
			}
		}

		sr := toStreamResponse(p, metaMap)
		out = append(out, sr)
	}

	// Refresh active streams metric
	metrics.ActiveStreams.Set(float64(len(paths)))
	for _, p := range paths {
		metrics.StreamReaders.WithLabelValues(p.Name).Set(float64(len(p.Readers)))
	}

	if out == nil {
		out = []streamResponse{}
	}
	jsonOK(w, out)
}

func (s *Server) handleGetStream(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	claims := claimsFromCtx(r)

	if claims.Role != string(dbpkg.RoleAdmin) {
		ok, _ := dbpkg.CheckAccess(s.db, claims.UserID, name, dbpkg.ACLActionRead)
		if !ok {
			jsonErr(w, http.StatusForbidden, "access denied")
			return
		}
	}

	paths, err := s.mtx.ListAllPaths()
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "mediamtx unreachable")
		return
	}

	metaMap, _ := dbpkg.ListStreamMeta(s.db)

	for _, p := range paths {
		if p.Name == name {
			jsonOK(w, toStreamResponse(p, metaMap))
			return
		}
	}
	jsonErr(w, http.StatusNotFound, "stream not found")
}

func (s *Server) handleStreamURLs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	claims := claimsFromCtx(r)

	if claims.Role != string(dbpkg.RoleAdmin) {
		ok, _ := dbpkg.CheckAccess(s.db, claims.UserID, name, dbpkg.ACLActionRead)
		if !ok {
			jsonErr(w, http.StatusForbidden, "access denied")
			return
		}
	}

	user, err := dbpkg.GetUserByID(s.db, claims.UserID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "user lookup failed")
		return
	}

	host := s.mediamtxPublicHostFor(r)
	userinfo := ""
	streamToken := ""
	if user.StreamTokenHash != nil {
		streamToken = *user.StreamTokenHash
		userinfo = url.UserPassword(user.Username, streamToken).String() + "@"
	}

	sc := schemeFor(r)
	hlsURL := fmt.Sprintf("%s://%s%s:%d/%s/index.m3u8", sc, userinfo, host, s.cfg.MediaMTX.HLSPort, name)
	rtmpURL := fmt.Sprintf("rtmp://%s:%d/%s", host, s.cfg.MediaMTX.RTMPPort, name)
	if streamToken != "" {
		rtmpURL = fmt.Sprintf("rtmp://%s%s:%d/%s", userinfo, host, s.cfg.MediaMTX.RTMPPort, name)
	}

	srtStreamID := "publish:" + name
	if streamToken != "" {
		srtStreamID = fmt.Sprintf("publish:%s:%s:%s", name, user.Username, streamToken)
	}
	srtURL := fmt.Sprintf("srt://%s:%d?streamid=%s", host, s.cfg.MediaMTX.SRTPort, srtStreamID)

	// Assume publish stream unless the config explicitly shows a pull source.
	// "publisher" is mediamtx's explicit keyword for push-only paths; empty
	// string means the same when no source is set via the API.
	isPublish := true
	if cfg, err := s.mtx.GetConfigPath(name); err == nil {
		isPublish = cfg.Source == "" || cfg.Source == "publisher"
	}

	jsonOK(w, streamURLsResponse{
		RTSP:            fmt.Sprintf("rtsp://%s%s:%d/%s", userinfo, host, s.cfg.MediaMTX.RTSPPort, name),
		HLS:             hlsURL,
		WebRTC:          fmt.Sprintf("%s://%s:%d/%s", sc, host, s.cfg.MediaMTX.WebRTCPort, name),
		RTMP:            rtmpURL,
		SRT:             srtURL,
		StreamToken:     streamToken,
		Username:        user.Username,
		IsPublishStream: isPublish,
	})
}

func (s *Server) handleCreateStream(w http.ResponseWriter, r *http.Request) {
	var req createStreamRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		jsonErr(w, http.StatusBadRequest, "name is required")
		return
	}

	cfg := mediamtx.PathConfig{
		Name:           req.Name,
		Source:         req.Source,
		SourceOnDemand: req.SourceOnDemand,
		Record:         req.Record,
		MaxReaders:     req.MaxReaders,
	}
	if err := s.mtx.AddConfigPath(req.Name, cfg); err != nil {
		jsonErr(w, http.StatusBadGateway, err.Error())
		return
	}

	if req.Description != "" {
		_ = dbpkg.UpsertStreamMeta(s.db, req.Name, req.Description)
	}

	jsonCreated(w, map[string]string{"name": req.Name})
}

func (s *Server) handleUpdateStream(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req createStreamRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request")
		return
	}

	cfg := mediamtx.PathConfig{
		Source:         req.Source,
		SourceOnDemand: req.SourceOnDemand,
		Record:         req.Record,
		MaxReaders:     req.MaxReaders,
	}
	if err := s.mtx.PatchConfigPath(name, cfg); err != nil {
		jsonErr(w, http.StatusBadGateway, err.Error())
		return
	}

	_ = dbpkg.UpsertStreamMeta(s.db, name, req.Description)
	noContent(w)
}

func (s *Server) handleDeleteStream(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.mtx.DeleteConfigPath(name); err != nil {
		jsonErr(w, http.StatusBadGateway, err.Error())
		return
	}
	_ = dbpkg.DeleteStreamMeta(s.db, name)
	noContent(w)
}

// schemeFor returns "https" when the request arrived over TLS or via a TLS proxy
// (X-Forwarded-Proto: https set by Traefik/Caddy), otherwise "http".
func schemeFor(r *http.Request) string {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

// mediamtxPublicHost returns the configured public host (or the API address host).
func (s *Server) mediamtxPublicHost() string {
	if s.cfg.MediaMTX.PublicHost != "" {
		return s.cfg.MediaMTX.PublicHost
	}
	// Extract host from API address
	u, err := url.Parse(s.cfg.MediaMTX.APIAddress)
	if err == nil {
		return u.Hostname()
	}
	return "localhost"
}

// mediamtxPublicHostFor returns the public host, falling back to the incoming
// request's hostname when the configured host is localhost/127.0.0.1.
// This ensures stream URLs work when the UI is accessed remotely without
// MEDIAMTX_UI_MEDIAMTX_PUBLIC_HOST being set.
func (s *Server) mediamtxPublicHostFor(r *http.Request) string {
	h := s.mediamtxPublicHost()
	if h == "localhost" || h == "127.0.0.1" {
		// r.Host may be "ip:port" or just "ip"
		reqHost := r.Host
		if host, _, err := net.SplitHostPort(reqHost); err == nil {
			reqHost = host
		}
		if reqHost != "" {
			return reqHost
		}
	}
	return h
}

func toStreamResponse(p mediamtx.PathItem, metaMap map[string]*dbpkg.StreamMeta) streamResponse {
	sr := streamResponse{
		Name:          p.Name,
		Ready:         p.Ready,
		Tracks:        p.Tracks,
		Readers:       len(p.Readers),
		BytesReceived: p.BytesReceived,
		BytesSent:     p.BytesSent,
	}
	if sr.Tracks == nil {
		sr.Tracks = []string{}
	}
	if p.Source != nil {
		sr.Source = p.Source.Type
	}
	if m, ok := metaMap[p.Name]; ok {
		sr.Description = m.Description
	}
	return sr
}
