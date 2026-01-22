package order

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CatalogProduct struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
}

type CatalogClient struct {
	BaseURL string
	Client  *http.Client
}

func NewCatalogClient(baseURL string) *CatalogClient {
	return &CatalogClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *CatalogClient) GetProduct(id string) (CatalogProduct, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/products/%s", c.BaseURL, id), nil)
	resp, err := c.Client.Do(req)
	if err != nil {
		return CatalogProduct{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return CatalogProduct{}, fmt.Errorf("catalog status %d", resp.StatusCode)
	}
	var p CatalogProduct
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return CatalogProduct{}, err
	}
	return p, nil
}
