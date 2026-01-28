package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"MiniStore/pkg/kit"
)

const maxBodyBytes = 1 << 20

type Server struct {
	Log   *zap.Logger
	Store UserStore
	JWT   *TokenMaker
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := s.Store.Ping(ctx); err != nil {
		if s.Log != nil {
			s.Log.Warn("readyz failed", zap.Error(err))
		}
		kit.WriteError(w, r, http.StatusServiceUnavailable, "not ready", nil)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer func() { _ = r.Body.Close() }()

	var req registerReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
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
	if err := s.Store.Create(r.Context(), req.Email, req.Password, "user", id); err != nil {
		switch err {
		case ErrEmailExists:
			kit.WriteError(w, r, http.StatusConflict, "email already exists", nil)
		default:
			if s.Log != nil {
				s.Log.Error("register failed", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		}
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
	defer func() { _ = r.Body.Close() }()

	var req loginReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		kit.WriteError(w, r, http.StatusBadRequest, "email/password required", nil)
		return
	}

	u, err := s.Store.Verify(r.Context(), req.Email, req.Password)
	if err != nil {
		kit.WriteError(w, r, http.StatusUnauthorized, "invalid credentials", nil)
		if s.Log != nil && !errors.Is(err, ErrInvalidCredentials) {
			s.Log.Warn("login verify failed", zap.Error(err))
		}
		return
	}

	tok, err := s.JWT.New(u.ID, u.Email, u.Role, 15*time.Minute)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("token issue", zap.Error(err))
		}
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
