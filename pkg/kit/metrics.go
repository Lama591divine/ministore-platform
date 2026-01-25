package kit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Reqs   *prometheus.CounterVec
	Latenc *prometheus.HistogramVec
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		Reqs: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"},
			[]string{"service", "method", "path", "status"},
		),
		Latenc: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP latency"},
			[]string{"service", "method", "path"},
		),
	}

	reg.MustRegister(m.Reqs, m.Latenc)
	return m
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (m *Metrics) Middleware(service string, pathLabel func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: 200}
			start := time.Now()

			next.ServeHTTP(sw, r)

			path := pathLabel(r)
			m.Latenc.WithLabelValues(service, r.Method, path).Observe(time.Since(start).Seconds())
			m.Reqs.WithLabelValues(service, r.Method, path, strconv.Itoa(sw.status)).Inc()
		})
	}
}
