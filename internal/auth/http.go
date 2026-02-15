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

func (s *Server) warn(msg string, err error) {
	if s.Log != nil {
		s.Log.Warn(msg, zap.Error(err))
	}
}

func (s *Server) err(msg string, err error) {
	if s.Log != nil {
		s.Log.Error(msg, zap.Error(err))
	}
}

func badRequest(w http.ResponseWriter, r *http.Request, msg string, meta map[string]any) {
	kit.WriteError(w, r, http.StatusBadRequest, msg, meta)
}

func unauthorized(w http.ResponseWriter, r *http.Request, msg string) {
	kit.WriteError(w, r, http.StatusUnauthorized, msg, nil)
}

func conflict(w http.ResponseWriter, r *http.Request, msg string) {
	kit.WriteError(w, r, http.StatusConflict, msg, nil)
}

func serverError(w http.ResponseWriter, r *http.Request) {
	kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer func() { _ = r.Body.Close() }()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("extra data after json object")
	}
	return nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func normalizePassword(s string) string {
	return strings.TrimSpace(s)
}

func bearerToken(r *http.Request) (string, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(authz, "Bearer ")), true
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := s.Store.Ping(ctx); err != nil {
		s.warn("readyz failed", err)
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
	var req registerReq
	if err := decodeJSON(w, r, &req); err != nil {
		badRequest(w, r, "bad json", nil)
		return
	}

	req.Email = normalizeEmail(req.Email)
	req.Password = normalizePassword(req.Password)

	if req.Email == "" || req.Password == "" {
		badRequest(w, r, "email/password required", nil)
		return
	}
	if len(req.Password) < 8 {
		badRequest(w, r, "password too short", map[string]any{"min_len": 8})
		return
	}

	id := "u_" + uuid.NewString()
	if err := s.Store.Create(r.Context(), req.Email, req.Password, "user", id); err != nil {
		if errors.Is(err, ErrEmailExists) {
			conflict(w, r, "email already exists")
			return
		}
		s.err("register failed", err)
		serverError(w, r)
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
	var req loginReq
	if err := decodeJSON(w, r, &req); err != nil {
		badRequest(w, r, "bad json", nil)
		return
	}

	req.Email = normalizeEmail(req.Email)
	req.Password = normalizePassword(req.Password)

	if req.Email == "" || req.Password == "" {
		badRequest(w, r, "email/password required", nil)
		return
	}

	u, err := s.Store.Verify(r.Context(), req.Email, req.Password)
	if err != nil {
		unauthorized(w, r, "invalid credentials")
		if !errors.Is(err, ErrInvalidCredentials) {
			s.warn("login verify failed", err)
		}
		return
	}

	tok, err := s.JWT.New(u.ID, u.Email, u.Role, 15*time.Minute)
	if err != nil {
		s.err("token issue", err)
		serverError(w, r)
		return
	}

	kit.WriteJSON(w, http.StatusOK, loginResp{AccessToken: tok})
}

func (s *Server) handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	tok, ok := bearerToken(r)
	if !ok || tok == "" {
		unauthorized(w, r, "missing token")
		return
	}

	claims, err := s.JWT.Parse(tok)
	if err != nil {
		unauthorized(w, r, "invalid token")
		return
	}

	kit.WriteJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}
