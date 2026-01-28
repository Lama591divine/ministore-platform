//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"
)

var baseURL = getenv("E2E_BASE_URL", "http://localhost:8080")

func TestSystem_E2E_WithDB(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	waitReady(t, ctx, baseURL+"/readyz")

	email := fmt.Sprintf("user_%d_%d@example.com", time.Now().Unix(), rand.Intn(100000))
	pass := "password123!"

	doJSON(t, http.MethodPost, baseURL+"/auth/register", map[string]any{
		"email":    email,
		"password": pass,
	}, nil, 201)

	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	doJSON(t, http.MethodPost, baseURL+"/auth/login", map[string]any{
		"email":    email,
		"password": pass,
	}, &loginResp, 200)
	if loginResp.AccessToken == "" {
		t.Fatalf("empty access_token")
	}

	var products []map[string]any
	doJSONAuth(t, http.MethodGet, baseURL+"/products", loginResp.AccessToken, nil, &products, 200)
	if len(products) == 0 {
		t.Fatalf("expected non-empty products")
	}

	pid, _ := products[0]["id"].(string)
	if pid == "" {
		t.Fatalf("product id missing in response: %#v", products[0])
	}

	var created map[string]any
	doJSONAuth(t, http.MethodPost, baseURL+"/orders", loginResp.AccessToken, map[string]any{
		"items": []map[string]any{
			{"product_id": pid, "qty": 2},
		},
	}, &created, 201)

	orderID, _ := created["id"].(string)
	if orderID == "" {
		t.Fatalf("order id missing: %#v", created)
	}

	var got map[string]any
	doJSONAuth(t, http.MethodGet, baseURL+"/orders/"+orderID, loginResp.AccessToken, nil, &got, 200)

	if os.Getenv("E2E_RESTART_ORDER") == "1" {
		restartOrderContainer(t, ctx)
		waitReady(t, ctx, baseURL+"/readyz")
		doJSONAuth(t, http.MethodGet, baseURL+"/orders/"+orderID, loginResp.AccessToken, nil, &got, 200)
	}
}

func waitReady(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil && resp != nil && resp.StatusCode == 200 {
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

func doJSON(t *testing.T, method, url string, body any, out any, want int) {
	t.Helper()
	doJSONAuth(t, method, url, "", body, out, want)
}

func doJSONAuth(t *testing.T, method, url, token string, body any, out any, want int) {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != want {
		t.Fatalf("%s %s: status=%d want=%d", method, url, resp.StatusCode, want)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
