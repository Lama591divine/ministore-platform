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
		cutoff := now.Add(-l.window)

		limited := l.recordAndCheck(ip, now, cutoff)
		if limited {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *IPRateLimiter) recordAndCheck(ip string, now, cutoff time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	ts := l.hits[ip]
	ts = prune(ts, cutoff)

	if len(ts) >= l.limit {
		l.hits[ip] = ts
		return true
	}

	l.hits[ip] = append(ts, now)
	return false
}

func prune(ts []time.Time, cutoff time.Time) []time.Time {
	n := 0
	for _, t := range ts {
		if t.After(cutoff) {
			ts[n] = t
			n++
		}
	}
	return ts[:n]
}

func clientIP(r *http.Request) string {
	if ip := firstForwardedFor(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func firstForwardedFor(xff string) string {
	if xff == "" {
		return ""
	}

	p := strings.Split(xff, ",")
	if len(p) == 0 {
		return ""
	}

	return strings.TrimSpace(p[0])
}
