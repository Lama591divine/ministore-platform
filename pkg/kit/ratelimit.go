package kit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type IPRateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func NewIPRateLimiter(limit int, windowSeconds int) *IPRateLimiter {
	return &IPRateLimiter{
		limit:  limit,
		window: time.Duration(windowSeconds) * time.Second,
		hits:   make(map[string][]time.Time),
	}
}

func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		now := time.Now()
		cut := now.Add(-l.window)

		l.mu.Lock()
		ts := l.hits[ip]

		n := 0
		for _, t := range ts {
			if t.After(cut) {
				ts[n] = t
				n++
			}
		}
		ts = ts[:n]

		if len(ts) >= l.limit {
			l.hits[ip] = ts
			l.mu.Unlock()
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		ts = append(ts, now)
		l.hits[ip] = ts
		l.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
