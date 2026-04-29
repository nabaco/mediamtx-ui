package api

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	dbpkg "mediamtx-ui/internal/db"
	"mediamtx-ui/internal/metrics"
)

func userAndPassFromQuery(query string) (user, pass string) {
	q, err := url.ParseQuery(query)
	if err != nil {
		return "", ""
	}
	return q.Get("user"), q.Get("password")
}

// mediamtxAuthRequest is the payload mediamtx POSTs to the auth callback.
// See: https://github.com/bluenviron/mediamtx#authentication
type mediamtxAuthRequest struct {
	User       string `json:"user"`
	Password   string `json:"password"`
	RemoteAddr string `json:"remoteAddr"`
	Action     string `json:"action"` // "read" | "publish"
	Path       string `json:"path"`
	Protocol   string `json:"protocol"` // "rtsp" | "rtmp" | "hls" | "webrtc" | "srt"
	ID         string `json:"id"`
	Query      string `json:"query"`
}

func (s *Server) handleMediaMTXAuth(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req mediamtxAuthRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	slog.Debug("mediamtx auth request", "raw", string(body),
		"user", req.User, "path", req.Path, "action", req.Action,
		"protocol", req.Protocol, "query", req.Query)

	allowed, username := s.checkMediaMTXAuth(req)

	// Write audit log asynchronously so we don't block the response
	go func() {
		_ = dbpkg.WriteAuditLog(s.db, dbpkg.AuditEntry{
			Username:   username,
			StreamPath: req.Path,
			Action:     req.Action,
			Protocol:   req.Protocol,
			RemoteAddr: req.RemoteAddr,
			Allowed:    allowed,
		})
	}()

	action := req.Action
	if action == "" {
		action = "read"
	}
	allowedStr := "false"
	if allowed {
		allowedStr = "true"
	}
	metrics.AuthCallbacks.WithLabelValues(action, allowedStr).Inc()

	if allowed {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

func (s *Server) checkMediaMTXAuth(req mediamtxAuthRequest) (allowed bool, resolvedUsername string) {
	if req.User == "" && req.Query != "" {
		req.User, req.Password = userAndPassFromQuery(req.Query)
	}

	if req.User == "" {
		return false, "anonymous"
	}

	user, err := dbpkg.GetUserByUsername(s.db, req.User)
	if err != nil || !user.Enabled {
		return false, req.User
	}

	// Verify stream token (stored plaintext)
	if user.StreamTokenHash == nil || *user.StreamTokenHash != req.Password {
		return false, req.User
	}

	// Admins have implicit access to all streams — no ACL check needed.
	if user.Role == dbpkg.RoleAdmin {
		return true, req.User
	}

	// Check ACL for regular users
	action := dbpkg.ACLActionRead
	if req.Action == "publish" {
		action = dbpkg.ACLActionPublish
	}

	ok, err := dbpkg.CheckAccess(s.db, user.ID, req.Path, action)
	if err != nil || !ok {
		return false, req.User
	}

	return true, req.User
}
