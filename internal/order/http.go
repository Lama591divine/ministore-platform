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

const (
	maxCreateBody = 1 << 20
)

func (s *Server) CreateHandler() http.HandlerFunc { return s.create }
func (s *Server) GetHandler() http.HandlerFunc    { return s.get }

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		kit.WriteError(w, r, http.StatusUnauthorized, "no user", nil)
		return
	}

	req, err := decodeCreateRequest(w, r)
	if err != nil {
		kit.WriteError(w, r, http.StatusBadRequest, "bad json", nil)
		return
	}
	if len(req.Items) == 0 {
		kit.WriteError(w, r, http.StatusBadRequest, "items required", nil)
		return
	}

	totalCents, err := s.calculateTotal(r.Context(), req.Items)
	if err != nil {
		s.writeCreateError(w, r, err)
		return
	}

	o := Order{
		ID:         "o_" + uuid.NewString(),
		UserID:     u.ID,
		Items:      req.Items,
		TotalCents: totalCents,
		Status:     "NEW",
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.Store.Create(r.Context(), o); err != nil {
		if isTimeoutErr(err) {
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
	o, found, err := s.Store.Get(r.Context(), id)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("store get order failed", zap.Error(err), zap.String("order_id", id))
		}
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
		return
	}
	if !found {
		kit.WriteError(w, r, http.StatusNotFound, "not found", map[string]any{"id": id})
		return
	}
	if o.UserID != u.ID {
		kit.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}

	kit.WriteJSON(w, http.StatusOK, o)
}

func decodeCreateRequest(w http.ResponseWriter, r *http.Request) (createReq, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCreateBody)
	defer func() { _ = r.Body.Close() }()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req createReq
	if err := dec.Decode(&req); err != nil {
		return createReq{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return createReq{}, errors.New("extra data after json object")
	}

	return req, nil
}

var (
	errBadItem         = errors.New("bad item")
	errDuplicateItem   = errors.New("duplicate product_id")
	errInvalidProduct  = errors.New("invalid product_id")
	errCatalogDown     = errors.New("catalog unavailable")
	errCatalogUpstream = errors.New("catalog error")
	errTotalOverflow   = errors.New("total overflow")
)

func (s *Server) calculateTotal(ctx context.Context, items []Item) (int64, error) {
	seen := make(map[string]struct{}, len(items))
	var total int64

	for _, it := range items {
		pid := strings.TrimSpace(it.ProductID)
		if it.Qty <= 0 || pid == "" {
			return 0, errBadItem
		}
		if _, dup := seen[pid]; dup {
			return 0, errDuplicateItem
		}
		seen[pid] = struct{}{}

		p, err := s.Catalog.GetProduct(ctx, pid)
		if err != nil {
			switch err {
			case ErrCatalogNotFound:
				return 0, errInvalidProduct
			case ErrCatalogUnavailable:
				return 0, errCatalogDown
			default:
				if s.Log != nil {
					s.Log.Warn("catalog error", zap.Error(err), zap.String("product_id", pid))
				}
				return 0, errCatalogUpstream
			}
		}

		line := p.PriceCents * int64(it.Qty)
		if line < 0 || total > math.MaxInt64-line {
			return 0, errTotalOverflow
		}
		total += line
	}

	return total, nil
}

func (s *Server) writeCreateError(w http.ResponseWriter, r *http.Request, err error) {
	switch err {
	case errBadItem:
		kit.WriteError(w, r, http.StatusBadRequest, "bad item", nil)
	case errDuplicateItem:
		kit.WriteError(w, r, http.StatusBadRequest, "duplicate product_id", nil)
	case errInvalidProduct:
		kit.WriteError(w, r, http.StatusBadRequest, "invalid product_id", nil)
	case errCatalogDown:
		kit.WriteError(w, r, http.StatusServiceUnavailable, "catalog unavailable", nil)
	case errCatalogUpstream:
		kit.WriteError(w, r, http.StatusBadGateway, "catalog error", nil)
	case errTotalOverflow:
		kit.WriteError(w, r, http.StatusBadRequest, "total overflow", nil)
	default:
		kit.WriteError(w, r, http.StatusInternalServerError, "server error", nil)
	}
}

func isTimeoutErr(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
