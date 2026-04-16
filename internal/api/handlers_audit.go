package api

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"

	dbpkg "mediamtx-ui/internal/db"
)

type auditEntryResponse struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	StreamPath string `json:"streamPath"`
	Action     string `json:"action"`
	Protocol   string `json:"protocol"`
	RemoteAddr string `json:"remoteAddr"`
	Allowed    bool   `json:"allowed"`
	CreatedAt  string `json:"createdAt"`
}

func parseAuditFilter(q url.Values) dbpkg.AuditFilter {
	f := dbpkg.AuditFilter{
		Username:   q.Get("username"),
		StreamPath: q.Get("stream"),
	}
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 {
		f.Limit = l
	} else {
		f.Limit = 50
	}
	if o, err := strconv.Atoi(q.Get("offset")); err == nil {
		f.Offset = o
	}
	return f
}

func parseAuditQuery(db *sql.DB, f dbpkg.AuditFilter) ([]auditEntryResponse, int, error) {
	entries, total, err := dbpkg.ListAuditLog(db, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]auditEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = auditEntryResponse{
			ID:         e.ID,
			Username:   e.Username,
			StreamPath: e.StreamPath,
			Action:     e.Action,
			Protocol:   e.Protocol,
			RemoteAddr: e.RemoteAddr,
			Allowed:    e.Allowed,
			CreatedAt:  e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return out, total, nil
}

// handleListAuditLog is the actual route handler (registered in router.go)
// The stub in handlers_system.go delegates here.
func auditLogHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := parseAuditFilter(r.URL.Query())
		entries, total, err := parseAuditQuery(s.db, filter)
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if entries == nil {
			entries = []auditEntryResponse{}
		}
		jsonOK(w, map[string]any{
			"total":   total,
			"entries": entries,
		})
	}
}
