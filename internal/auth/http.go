package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"MiniStore/pkg/kit"
)

const maxBodyBytes = 1 << 20

type Server struct {
	Log   *zap.Logger
	Store *Store
	JWT   *TokenMaker
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Post("/auth/register", s.handleRegister)
	r.Post("/auth/login", s.handleLogin)
	r.Get("/auth/whoami", s.handleWhoAmI)

	return r
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req registerReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", map[string]any{"cause": err.Error()})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		kit.WriteError(w, r, http.StatusBadRequest, "email/password required", nil)
		return
	}
	if len(req.Password) < 8 {
		kit.WriteError(w, r, http.StatusBadRequest, "password too short", map[string]any{"min_len": 8})
		return
	}

	id := "u_" + uuid.NewString()

	if err := s.Store.Create(req.Email, req.Password, "user", id); err != nil {
		kit.WriteError(w, r, http.StatusConflict, err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResp struct {
	AccessToken string `json:"access_token"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req loginReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", map[string]any{"cause": err.Error()})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		kit.WriteError(w, r, http.StatusBadRequest, "email/password required", nil)
		return
	}

	u, err := s.Store.Verify(req.Email, req.Password)
	if err != nil {
		kit.WriteError(w, r, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	tok, err := s.JWT.New(u.ID, u.Email, u.Role, 15*time.Minute)
	if err != nil {
		s.Log.Error("token issue", zap.Error(err))
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		return
	}

	kit.WriteJSON(w, http.StatusOK, loginResp{AccessToken: tok})
}

func (s *Server) handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		kit.WriteError(w, r, http.StatusUnauthorized, "missing token", nil)
		return
	}

	claims, err := s.JWT.Parse(strings.TrimPrefix(authz, "Bearer "))
	if err != nil {
		kit.WriteError(w, r, http.StatusUnauthorized, "invalid token", nil)
		return
	}

	kit.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}
