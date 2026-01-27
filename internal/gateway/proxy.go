package gateway

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
)

type ctxKey string

const (
	userIDKey   ctxKey = "user_id"
	userRoleKey ctxKey = "user_role"
)

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userRoleKey).(string)
	return v, ok
}

func AuthJWT(jwt *auth.TokenMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") {
				kit.WriteError(w, r, http.StatusUnauthorized, "missing token", nil)
				return
			}
			claims, err := jwt.Parse(strings.TrimPrefix(authz, "Bearer "))
			if err != nil {
				kit.WriteError(w, r, http.StatusUnauthorized, "invalid token", nil)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, userRoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewReverseProxy(target string, log *zap.Logger) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := httputil.NewSingleHostReverseProxy(u)

	p.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if log != nil {
			log.Warn("proxy error",
				zap.String("target", target),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Error(err),
			)
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			kit.WriteError(w, r, http.StatusGatewayTimeout, "upstream timeout", nil)
			return
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			kit.WriteError(w, r, http.StatusGatewayTimeout, "upstream timeout", nil)
			return
		}
		kit.WriteError(w, r, http.StatusBadGateway, "bad gateway", nil)
	}

	p.ModifyResponse = func(resp *http.Response) error {
		if resp != nil && resp.StatusCode >= 500 && log != nil && resp.Request != nil && resp.Request.URL != nil {
			log.Warn("upstream 5xx",
				zap.String("target", target),
				zap.Int("status", resp.StatusCode),
				zap.String("path", resp.Request.URL.Path),
			)
		}
		return nil
	}

	return p, nil
}

func InjectHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-Id")
		r.Header.Del("X-User-Role")

		if uid, ok := UserIDFromContext(r.Context()); ok && uid != "" {
			r.Header.Set("X-User-Id", uid)
		}
		if role, ok := UserRoleFromContext(r.Context()); ok && role != "" {
			r.Header.Set("X-User-Role", role)
		}

		next.ServeHTTP(w, r)
	})
}
