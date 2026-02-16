package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CatalogProduct struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
}

var (
	ErrCatalogNotFound    = errors.New("catalog product not found")
	ErrCatalogBadStatus   = errors.New("catalog bad status")
	ErrCatalogUnavailable = errors.New("catalog unavailable")
)

const (
	catalogTimeout = 3 * time.Second
)

type CatalogClient struct {
	BaseURL string
	Client  *http.Client
}

func NewCatalogClient(baseURL string) *CatalogClient {
	baseURL = normalizeBaseURL(baseURL)

	return &CatalogClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: catalogTimeout},
	}
}

func (c *CatalogClient) GetProduct(ctx context.Context, id string) (CatalogProduct, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/products/"+id, nil)
	if err != nil {
		return CatalogProduct{}, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return CatalogProduct{}, mapCatalogError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return CatalogProduct{}, ErrCatalogNotFound
	}

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return CatalogProduct{}, fmt.Errorf("%w: status=%d", ErrCatalogBadStatus, resp.StatusCode)
	}

	var p CatalogProduct
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return CatalogProduct{}, err
	}

	return p, nil
}

func normalizeBaseURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err == nil && u.Scheme != "" && u.Host != "" {
		return strings.TrimRight(baseURL, "/")
	}
	return baseURL
}

func mapCatalogError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrCatalogUnavailable
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return ErrCatalogUnavailable
	}
	return ErrCatalogUnavailable
}
