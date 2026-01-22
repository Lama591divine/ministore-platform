package order

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"MiniStore/pkg/kit"
)

type Server struct {
	Store   *Store
	Catalog *CatalogClient
}

type createReq struct {
	Items []Item `json:"items"`
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		kit.WriteError(w, r, http.StatusUnauthorized, "no user", nil)
		return
	}

	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}
	if len(req.Items) == 0 {
		kit.WriteError(w, r, http.StatusBadRequest, "items required", nil)
		return
	}

	var totalCents int64
	for _, it := range req.Items {
		if it.Qty <= 0 || strings.TrimSpace(it.ProductID) == "" {
			kit.WriteError(w, r, http.StatusBadRequest, "bad item", it)
			return
		}
		p, err := s.Catalog.GetProduct(it.ProductID)
		if err != nil {
			kit.WriteError(w, r, http.StatusBadRequest, "invalid product_id", map[string]any{"product_id": it.ProductID})
			return
		}
		totalCents += p.PriceCents * int64(it.Qty)
	}

	id := "o_" + u.ID + "_" + time.Now().Format("20060102150405")
	o := Order{
		ID:         id,
		UserID:     u.ID,
		Items:      req.Items,
		TotalCents: totalCents,
		Status:     "NEW",
		CreatedAt:  time.Now(),
	}
	s.Store.Put(o)

	kit.WriteJSON(w, http.StatusOK, o)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, ok := s.Store.Get(id)
	if !ok {
		kit.WriteError(w, r, http.StatusNotFound, "not found", map[string]any{"id": id})
		return
	}
	kit.WriteJSON(w, http.StatusOK, o)
}

func (s *Server) CreateHandler() http.HandlerFunc { return s.create }
func (s *Server) GetHandler() http.HandlerFunc    { return s.get }
