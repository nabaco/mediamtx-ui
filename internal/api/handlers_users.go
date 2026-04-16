package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	dbpkg "mediamtx-ui/internal/db"
)

type userResponse struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	Enabled      bool   `json:"enabled"`
	HasToken     bool   `json:"hasToken"`
	CreatedAt    string `json:"createdAt"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Password *string `json:"password"`
	Role     *string `json:"role"`
	Enabled  *bool   `json:"enabled"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := dbpkg.ListUsers(s.db)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]userResponse, len(users))
	for i, u := range users {
		out[i] = toUserResponse(u)
	}
	jsonOK(w, out)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil || req.Username == "" || req.Password == "" {
		jsonErr(w, http.StatusBadRequest, "username and password required")
		return
	}

	role := dbpkg.RoleUser
	if req.Role == string(dbpkg.RoleAdmin) {
		role = dbpkg.RoleAdmin
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "password hashing failed")
		return
	}

	user, err := dbpkg.CreateUser(s.db, req.Username, string(hash), role)
	if err != nil {
		if err == dbpkg.ErrConflict {
			jsonErr(w, http.StatusConflict, "username already exists")
			return
		}
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonCreated(w, toUserResponse(user))
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, err := dbpkg.GetUserByID(s.db, id)
	if err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, toUserResponse(user))
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request")
		return
	}

	p := dbpkg.UpdateUserParams{
		Enabled: req.Enabled,
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, "password hashing failed")
			return
		}
		h := string(hash)
		p.PasswordHash = &h
	}
	if req.Role != nil {
		role := dbpkg.Role(*req.Role)
		p.Role = &role
	}

	user, err := dbpkg.UpdateUser(s.db, id, p)
	if err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, toUserResponse(user))
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Prevent self-deletion
	claims := claimsFromCtx(r)
	if claims.UserID == id {
		jsonErr(w, http.StatusBadRequest, "cannot delete your own account")
		return
	}

	if err := dbpkg.DeleteUser(s.db, id); err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	} else if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	noContent(w)
}

func (s *Server) handleRegenerateStreamToken(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Generate a cryptographically random 32-byte token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		jsonErr(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	token := base64.URLEncoding.EncodeToString(raw)

	// Stream tokens are stored plaintext — they are already distributed in RTSP URLs
	// and need to be retrievable for WHEP/browser playback.
	_, err = dbpkg.UpdateUser(s.db, id, dbpkg.UpdateUserParams{StreamTokenHash: &token})
	if err == dbpkg.ErrNotFound {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the raw token once — caller must save it
	jsonOK(w, map[string]string{"token": token})
}

func toUserResponse(u *dbpkg.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Role:      string(u.Role),
		Enabled:   u.Enabled,
		HasToken:  u.StreamTokenHash != nil,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func pathParamID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}
