package api

import (
	"bytes"
	"crypto/subtle"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"

	dbpkg "mediamtx-ui/internal/db"
	"mediamtx-ui/internal/metrics"
)

func userAndPassFromQuery(query string) (user, pass string) {
	q, err := url.ParseQuery(query)
	if err != nil {
		return "", ""
	}
	// mediamtx RTMP convention is ?user=&pass= (note "pass", not "password").
	return q.Get("user"), q.Get("pass")
}

// mediamtxAuthRequest is the payload mediamtx POSTs to the auth callback.
// See: https://github.com/bluenviron/mediamtx#authentication
// Note: mediamtx v1 uses "ip" (not "remoteAddr") and adds a "token" field.
type mediamtxAuthRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Token    string `json:"token"`  // query-param token (?token=xxx), used by HLS/WebRTC clients
	IP       string `json:"ip"`     // mediamtx v1 field name
	Action   string `json:"action"` // "read" | "publish"
	Path     string `json:"path"`
	Protocol string `json:"protocol"` // "rtsp" | "rtmp" | "hls" | "webrtc" | "srt"
	ID       string `json:"id"`
	Query    string `json:"query"`
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

	// Control-plane actions (api/metrics/pprof) arrive on every request the
	// UI makes to the mediamtx API; audit only the denials so the stream
	// access log is not flooded.
	audit := !isServiceAction(req.Action) || !allowed

	// Write audit log asynchronously so we don't block the response
	go func() {
		if !audit {
			return
		}
		_ = dbpkg.WriteAuditLog(s.db, dbpkg.AuditEntry{
			Username:   username,
			StreamPath: req.Path,
			Action:     req.Action,
			Protocol:   req.Protocol,
			RemoteAddr: req.IP,
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

// isServiceAction reports whether a mediamtx auth action targets the control
// plane rather than a stream. Since mediamtx 1.20.1 the default
// authHTTPExclude is empty, so these reach the HTTP callback too.
func isServiceAction(action string) bool {
	switch action {
	case "api", "metrics", "pprof":
		return true
	}
	return false
}

// isStreamAction reports whether action is one we map onto stream ACLs.
// "playback" (the recordings playback server) is treated as read.
func isStreamAction(action string) bool {
	switch action {
	case "", "read", "publish", "playback":
		return true
	}
	return false
}

// checkServiceAuth allows control-plane actions only for the mediamtx API
// service account mediamtx-ui itself uses (mediamtx.api_user / api_pass).
// UI users, admins included, never get raw mediamtx API access this way.
func (s *Server) checkServiceAuth(req mediamtxAuthRequest) bool {
	u, p := s.cfg.MediaMTX.APIUser, s.cfg.MediaMTX.APIPass
	if u == "" || p == "" {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(req.User), []byte(u)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(p)) == 1
	return userOK && passOK
}

func (s *Server) checkMediaMTXAuth(req mediamtxAuthRequest) (allowed bool, resolvedUsername string) {
	if isServiceAction(req.Action) {
		if req.User == "" {
			return false, "anonymous"
		}
		return s.checkServiceAuth(req), req.User
	}
	if !isStreamAction(req.Action) {
		return false, req.User
	}

	// Extract credentials from query string if not in top-level fields.
	if req.User == "" && req.Query != "" {
		req.User, req.Password = userAndPassFromQuery(req.Query)
	}
	// Some clients (HLS/WebRTC) pass ?token=xxx without a username.
	if req.Token != "" && req.Password == "" {
		req.Password = req.Token
	}

	if req.User == "" {
		return false, "anonymous"
	}

	user, err := dbpkg.GetUserByUsername(s.db, req.User)
	if err != nil || !user.Enabled {
		return false, req.User
	}

	// JWT auth: browser player sends the UI JWT as Basic auth password.
	if looksLikeJWT(req.Password) {
		claims, err := s.jwt.Verify(req.Password)
		if err != nil || claims.Username != req.User {
			return false, req.User
		}
		if user.Role == dbpkg.RoleAdmin {
			return true, req.User
		}
		action := dbpkg.ACLActionRead
		if req.Action == "publish" {
			action = dbpkg.ACLActionPublish
		}
		ok, _ := dbpkg.CheckAccess(s.db, user.ID, req.Path, action)
		return ok, req.User
	}

	// Standard bcrypt password auth (VLC, OBS, ffmpeg, external clients).
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return false, req.User
	}

	if user.Role == dbpkg.RoleAdmin {
		return true, req.User
	}

	action := dbpkg.ACLActionRead
	if req.Action == "publish" {
		action = dbpkg.ACLActionPublish
	}
	ok, _ := dbpkg.CheckAccess(s.db, user.ID, req.Path, action)
	return ok, req.User
}

// looksLikeJWT returns true for HS256 JWTs (3-part dot-separated, eyJ prefix).
func looksLikeJWT(s string) bool {
	return strings.Count(s, ".") == 2 && strings.HasPrefix(s, "eyJ")
}
