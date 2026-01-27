package order

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"MiniStore/pkg/kit"
)

type Server struct {
	Store   Store
	Catalog *CatalogClient
	Log     *zap.Logger
}

type createReq struct {
	Items []Item `json:"items"`
}

const maxCreateBody = 1 << 20

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		kit.WriteError(w, r, http.StatusUnauthorized, "no user", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCreateBody)
	defer func() { _ = r.Body.Close() }()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req createReq
	if err := dec.Decode(&req); err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}

	if len(req.Items) == 0 {
		kit.WriteError(w, r, http.StatusBadRequest, "items required", nil)
		return
	}

	seen := map[string]struct{}{}
	var totalCents int64

	for _, it := range req.Items {
		pid := strings.TrimSpace(it.ProductID)
		if it.Qty <= 0 || pid == "" {
			kit.WriteError(w, r, http.StatusBadRequest, "bad item", map[string]any{"product_id": pid, "qty": it.Qty})
			return
		}
		if _, dup := seen[pid]; dup {
			kit.WriteError(w, r, http.StatusBadRequest, "duplicate product_id", map[string]any{"product_id": pid})
			return
		}
		seen[pid] = struct{}{}

		p, err := s.Catalog.GetProduct(r.Context(), pid)
		if err != nil {
			switch err {
			case ErrCatalogNotFound:
				kit.WriteError(w, r, http.StatusBadRequest, "invalid product_id", map[string]any{"product_id": pid})
			case ErrCatalogUnavailable:
				kit.WriteError(w, r, http.StatusServiceUnavailable, "catalog unavailable", nil)
			default:
				kit.WriteError(w, r, http.StatusBadGateway, "catalog error", nil)
				if s.Log != nil {
					s.Log.Warn("catalog error", zap.Error(err), zap.String("product_id", pid))
				}
			}
			return
		}

		line := p.PriceCents * int64(it.Qty)
		if line < 0 || totalCents > math.MaxInt64-line {
			kit.WriteError(w, r, http.StatusBadRequest, "total overflow", nil)
			return
		}
		totalCents += line
	}

	now := time.Now().UTC()
	id := "o_" + uuid.NewString()

	o := Order{
		ID:         id,
		UserID:     u.ID,
		Items:      req.Items,
		TotalCents: totalCents,
		Status:     "NEW",
		CreatedAt:  now,
	}

	if err := s.Store.Create(r.Context(), o); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			kit.WriteError(w, r, http.StatusGatewayTimeout, "timeout", nil)
			return
		}
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		return
	}

	kit.WriteJSON(w, http.StatusCreated, o)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		kit.WriteError(w, r, http.StatusUnauthorized, "no user", nil)
		return
	}

	id := chi.URLParam(r, "id")
	o, ok, err := s.Store.Get(r.Context(), id)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("store get order failed", zap.Error(err), zap.String("order_id", id))
		}
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		return
	}
	if !ok {
		kit.WriteError(w, r, http.StatusNotFound, "not found", map[string]any{"id": id})
		return
	}
	if o.UserID != u.ID {
		kit.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}

	kit.WriteJSON(w, http.StatusOK, o)
}

func (s *Server) CreateHandler() http.HandlerFunc { return s.create }
func (s *Server) GetHandler() http.HandlerFunc    { return s.get }
