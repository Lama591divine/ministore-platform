//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var baseURL = getenv("E2E_BASE_URL", "http://localhost:8080")

func TestSystem_E2E_WithDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	env := newE2EEnv(t, baseURL)
	env.WaitReady(ctx)

	mode := strings.ToLower(getenv("E2E_MODE", "create"))

	email := mustGetenv(t, "E2E_EMAIL")
	pass := mustGetenv(t, "E2E_PASSWORD")

	env.tryRegister(email, pass)

	token := env.Login(email, pass)
	if token == "" {
		t.Fatalf("empty access_token")
	}

	switch mode {
	case "create":
		orderID := createOrderFlow(t, env, token)

		fmt.Printf("E2E_ORDER_ID=%s\n", orderID)

	case "verify":
		orderID := mustGetenv(t, "E2E_ORDER_ID")

		env.WaitReady(ctx)

		got := env.GetOrder(token, orderID)
		if gotID := getString(got, "id"); gotID != orderID {
			t.Fatalf("after restart: got id=%q want=%q; body=%#v", gotID, orderID, got)
		}

	default:
		t.Fatalf("unknown E2E_MODE=%q", mode)
	}
}

func createOrderFlow(t *testing.T, env *e2eEnv, token string) string {
	t.Helper()

	products := env.Products(token)
	if len(products) == 0 {
		t.Fatalf("expected non-empty products")
	}

	pid := getString(products[0], "id")
	if pid == "" {
		t.Fatalf("product id missing in response: %#v", products[0])
	}

	orderID := env.CreateOrder(token, []itemReq{{ProductID: pid, Qty: 2}})
	if orderID == "" {
		t.Fatalf("order id missing")
	}

	got := env.GetOrder(token, orderID)
	if gotID := getString(got, "id"); gotID != orderID {
		t.Fatalf("got id=%q want=%q; body=%#v", gotID, orderID, got)
	}

	return orderID
}

type e2eEnv struct {
	t       *testing.T
	baseURL string
	client  *http.Client
}

func newE2EEnv(t *testing.T, baseURL string) *e2eEnv {
	t.Helper()
	return &e2eEnv{
		t:       t,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (e *e2eEnv) WaitReady(ctx context.Context) {
	e.t.Helper()
	waitReady(e.t, ctx, e.baseURL+"/readyz")
}

func (e *e2eEnv) tryRegister(email, password string) {
	e.t.Helper()

	req, err := newJSONRequest(http.MethodPost, e.baseURL+"/auth/register", "", map[string]any{
		"email":    email,
		"password": password,
	})
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return
	}
	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusBadRequest {
		return
	}

	raw, _ := io.ReadAll(resp.Body)
	e.t.Fatalf("register: status=%d body=%s", resp.StatusCode, string(raw))
}

func (e *e2eEnv) Login(email, password string) string {
	e.t.Helper()
	var out struct {
		AccessToken string `json:"access_token"`
	}
	e.mustJSON(http.MethodPost, "/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	}, &out, http.StatusOK)
	return out.AccessToken
}

func (e *e2eEnv) Products(token string) []map[string]any {
	e.t.Helper()
	var out []map[string]any
	e.mustJSON(http.MethodGet, "/products", token, nil, &out, http.StatusOK)
	return out
}

type itemReq struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

func (e *e2eEnv) CreateOrder(token string, items []itemReq) string {
	e.t.Helper()
	var out map[string]any
	e.mustJSON(http.MethodPost, "/orders", token, map[string]any{
		"items": items,
	}, &out, http.StatusCreated)
	return getString(out, "id")
}

func (e *e2eEnv) GetOrder(token, orderID string) map[string]any {
	e.t.Helper()
	var out map[string]any
	e.mustJSON(http.MethodGet, "/orders/"+orderID, token, nil, &out, http.StatusOK)
	return out
}

func (e *e2eEnv) mustJSON(method, path, token string, body any, out any, want int) {
	e.t.Helper()
	url := e.baseURL + path

	req, err := newJSONRequest(method, url, token, body)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != want {
		raw, _ := io.ReadAll(resp.Body)
		e.t.Fatalf("%s %s: status=%d want=%d body=%s", method, url, resp.StatusCode, want, string(raw))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			e.t.Fatalf("decode response: %v", err)
		}
	}
}

func newJSONRequest(method, url, token string, body any) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}
		r = &buf
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func waitReady(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("service not ready: %s", url)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustGetenv(t *testing.T, k string) string {
	t.Helper()
	v := os.Getenv(k)
	if v == "" {
		t.Fatalf("%s is empty", k)
	}
	return v
}
