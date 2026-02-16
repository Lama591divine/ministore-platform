package kit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelService = "service"
	labelMethod  = "method"
	labelPath    = "path"
	labelStatus  = "status"

	defaultStatusCode = http.StatusOK
)

type Metrics struct {
	Requests *prometheus.CounterVec
	Latency  *prometheus.HistogramVec
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		Requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{labelService, labelMethod, labelPath, labelStatus},
		),
		Latency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP latency",
			},
			[]string{labelService, labelMethod, labelPath},
		),
	}

	reg.MustRegister(m.Requests, m.Latency)
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

func (m *Metrics) Middleware(service string, pathLabel func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{
				ResponseWriter: w,
				status:         defaultStatusCode,
			}

			start := time.Now()
			next.ServeHTTP(sw, r)

			path := pathLabel(r)
			m.Latency.WithLabelValues(service, r.Method, path).
				Observe(time.Since(start).Seconds())

			m.Requests.WithLabelValues(service, r.Method, path, strconv.Itoa(sw.status)).
				Inc()
		})
	}
}
