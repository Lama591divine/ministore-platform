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

type CatalogClient struct {
	BaseURL string
	Client  *http.Client
}

func NewCatalogClient(baseURL string) *CatalogClient {
	if u, err := url.Parse(baseURL); err == nil && u.Scheme != "" && u.Host != "" {
		baseURL = strings.TrimRight(baseURL, "/")
	}
	return &CatalogClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *CatalogClient) GetProduct(ctx context.Context, id string) (CatalogProduct, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/products/%s", c.BaseURL, id), nil)
	if err != nil {
		return CatalogProduct{}, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return CatalogProduct{}, ErrCatalogUnavailable
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return CatalogProduct{}, ErrCatalogUnavailable
		}
		return CatalogProduct{}, ErrCatalogUnavailable
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return CatalogProduct{}, ErrCatalogNotFound
	default:
		_, _ = io.Copy(io.Discard, resp.Body)
		return CatalogProduct{}, fmt.Errorf("%w: status=%d", ErrCatalogBadStatus, resp.StatusCode)
	}

	var p CatalogProduct
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return CatalogProduct{}, err
	}
	return p, nil
}
