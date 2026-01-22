package catalog

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"MiniStore/pkg/kit"
)

type Server struct {
	Store *Store
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })

	r.Get("/products", s.list)
	r.Get("/products/{id}", s.get)

	return r
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	kit.WriteJSON(w, http.StatusOK, s.Store.List())
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, ok := s.Store.Get(id)
	if !ok {
		kit.WriteError(w, r, http.StatusNotFound, "not found", map[string]any{"id": id})
		return
	}
	kit.WriteJSON(w, http.StatusOK, p)
}
