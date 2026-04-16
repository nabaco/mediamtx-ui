package api

import (
	"context"
	"net/http"
	"strings"

	"mediamtx-ui/internal/auth"
	dbpkg "mediamtx-ui/internal/db"
)

type ctxKey string

const claimsKey ctxKey = "claims"

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			jsonErr(w, http.StatusUnauthorized, "missing token")
			return
		}
		claims, err := s.jwt.Verify(token)
		if err != nil {
			jsonErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r)
		if claims == nil || claims.Role != string(dbpkg.RoleAdmin) {
			jsonErr(w, http.StatusForbidden, "admin required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func claimsFromCtx(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return c
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return after
	}
	return ""
}
