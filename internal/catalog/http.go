package catalog

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"MiniStore/pkg/kit"
)

type Server struct {
	Store Store
	Log   *zap.Logger
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Get("/products", s.list)
	r.Get("/products/{id}", s.get)

	return r
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	products, err := s.Store.ListSortedByID(r.Context())
	if err != nil {
		if s.Log != nil {
			s.Log.Error("list products failed", zap.Error(err))
		}
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		return
	}
	kit.WriteJSON(w, http.StatusOK, products)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, ok, err := s.Store.Get(r.Context(), id)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("get product failed", zap.Error(err), zap.String("id", id))
		}
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		return
	}
	if !ok {
		kit.WriteError(w, r, http.StatusNotFound, "not found", map[string]any{"id": id})
		return
	}
	kit.WriteJSON(w, http.StatusOK, p)
}
