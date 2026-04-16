package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	dbpkg "mediamtx-ui/internal/db"
	"mediamtx-ui/internal/metrics"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type meResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request")
		return
	}

	user, err := dbpkg.GetUserByUsername(s.db, req.Username)
	if err != nil {
		metrics.LoginAttempts.WithLabelValues("failure").Inc()
		jsonErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !user.Enabled {
		metrics.LoginAttempts.WithLabelValues("failure").Inc()
		jsonErr(w, http.StatusUnauthorized, "account disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		metrics.LoginAttempts.WithLabelValues("failure").Inc()
		jsonErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := s.jwt.Sign(user.ID, user.Username, string(user.Role))
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	metrics.LoginAttempts.WithLabelValues("success").Inc()
	jsonOK(w, loginResponse{
		Token:    token,
		Username: user.Username,
		Role:     string(user.Role),
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
		jsonErr(w, http.StatusBadRequest, "currentPassword and newPassword are required")
		return
	}

	user, err := dbpkg.GetUserByID(s.db, claims.UserID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "user not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		jsonErr(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	hashStr := string(hash)
	if _, err := dbpkg.UpdateUser(s.db, claims.UserID, dbpkg.UpdateUserParams{PasswordHash: &hashStr}); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	noContent(w)
}

func (s *Server) handleGetStreamToken(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	user, err := dbpkg.GetUserByID(s.db, claims.UserID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "user not found")
		return
	}
	if user.StreamTokenHash == nil {
		jsonOK(w, map[string]any{"token": nil})
		return
	}
	jsonOK(w, map[string]string{"token": *user.StreamTokenHash})
}

func (s *Server) handleRegenerateOwnStreamToken(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		jsonErr(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	token := base64.URLEncoding.EncodeToString(raw)
	if _, err := dbpkg.UpdateUser(s.db, claims.UserID, dbpkg.UpdateUserParams{StreamTokenHash: &token}); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, map[string]string{"token": token})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	jsonOK(w, meResponse{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	})
}
