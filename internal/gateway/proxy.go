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

	bearerPrefix = "Bearer "
)

func AuthJWT(jwt *auth.TokenMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, ok := bearerToken(r)
			if !ok {
				kit.WriteError(w, r, http.StatusUnauthorized, "missing token", nil)
				return
			}

			claims, err := jwt.Parse(tok)
			if err != nil {
				kit.WriteError(w, r, http.StatusUnauthorized, "invalid token", nil)
				return
			}

			ctx := withAuthContext(r.Context(), claims.UserID, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, bearerPrefix) {
		return "", false
	}

	tok := strings.TrimSpace(strings.TrimPrefix(authz, bearerPrefix))
	if tok == "" {
		return "", false
	}

	return tok, true
}

func withAuthContext(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	return ctx
}

func NewReverseProxy(target string, log *zap.Logger) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := httputil.NewSingleHostReverseProxy(u)
	p.Transport = newProxyTransport()
	p.ErrorHandler = proxyErrorHandler(target, log)
	p.ModifyResponse = proxyModifyResponse(target, log)

	return p, nil
}

func newProxyTransport() *http.Transport {
	return &http.Transport{
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
}

func proxyErrorHandler(target string, log *zap.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		if log != nil {
			log.Warn("proxy error",
				zap.String("target", target),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Error(err),
			)
		}

		if isTimeoutErr(err) {
			kit.WriteError(w, r, http.StatusGatewayTimeout, "upstream timeout", nil)
			return
		}

		kit.WriteError(w, r, http.StatusBadGateway, "bad gateway", nil)
	}
}

func proxyModifyResponse(target string, log *zap.Logger) func(*http.Response) error {
	return func(resp *http.Response) error {
		if log == nil || resp == nil || resp.Request == nil || resp.Request.URL == nil {
			return nil
		}

		if resp.StatusCode >= 500 {
			log.Warn("upstream 5xx",
				zap.String("target", target),
				zap.Int("status", resp.StatusCode),
				zap.String("path", resp.Request.URL.Path),
			)
		}

		return nil
	}
}

func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
