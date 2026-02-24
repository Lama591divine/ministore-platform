package gateway_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"MiniStore/internal/auth"
	"MiniStore/internal/catalog"
	"MiniStore/internal/gateway"
	"MiniStore/internal/order"
)

const jwtSecret = "test-secret-32-chars-minimum-........"

type testEnv struct {
	Auth    *httptest.Server
	Catalog *httptest.Server
	Order   *httptest.Server
	GW      *httptest.Server
	Client  *http.Client
}

func newAuthTS(t *testing.T, jwtSecret string) *httptest.Server {
	t.Helper()

	s := &auth.Server{
		Log:   zap.NewNop(),
		Store: auth.NewMemStore(),
		JWT:   auth.NewTokenMaker(jwtSecret),
	}

	h := auth.NewHandler(s, auth.HTTPDeps{
		Log:     zap.NewNop(),
		Service: "auth",
	})

	return httptest.NewServer(h)
}

func newCatalogTS(t *testing.T) *httptest.Server {
	t.Helper()

	s := &catalog.Server{
		Store: catalog.NewMemStore(),
		Log:   zap.NewNop(),
	}

	h := catalog.NewHandler(s, catalog.HTTPDeps{
		Log:     zap.NewNop(),
		Service: "catalog",
	})

	return httptest.NewServer(h)
}

func newOrderTS(t *testing.T, jwtSecret, catalogURL string) *httptest.Server {
	t.Helper()

	s := &order.Server{
		Store:   order.NewMemStore(),
		Catalog: order.NewCatalogClient(catalogURL),
		Log:     zap.NewNop(),
	}

	h := order.NewHandler(s, order.HTTPDeps{
		Log:       zap.NewNop(),
		Service:   "order",
		JWTSecret: jwtSecret,
	})

	return httptest.NewServer(h)
}

func newGatewayTS(t *testing.T, jwtSecret, authURL, catalogURL, orderURL string) *httptest.Server {
	t.Helper()

	h, err := gateway.NewHandler(
		gateway.Deps{
			JWTSecret:  jwtSecret,
			AuthURL:    authURL,
			CatalogURL: catalogURL,
			OrderURL:   orderURL,
		},
		gateway.HTTPDeps{
			Log:     zap.NewNop(),
			Service: "gateway",
		},
	)
	if err != nil {
		t.Fatalf("gateway.NewHandler: %v", err)
	}

	return httptest.NewServer(h)
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()

	authTS := newAuthTS(t, jwtSecret)
	t.Cleanup(authTS.Close)

	catalogTS := newCatalogTS(t)
	t.Cleanup(catalogTS.Close)

	orderTS := newOrderTS(t, jwtSecret, catalogTS.URL)
	t.Cleanup(orderTS.Close)

	gwTS := newGatewayTS(t, jwtSecret, authTS.URL, catalogTS.URL, orderTS.URL)
	t.Cleanup(gwTS.Close)

	return testEnv{
		Auth:    authTS,
		Catalog: catalogTS,
		Order:   orderTS,
		GW:      gwTS,
		Client:  &http.Client{},
	}
}

func doJSON(t *testing.T, c *http.Client, method, url string, body any, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, raw
}

func mustStatus(t *testing.T, resp *http.Response, raw []byte, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, want, string(raw))
	}
}

func register(t *testing.T, env testEnv, email, password string) {
	t.Helper()
	resp, raw := doJSON(t, env.Client, http.MethodPost, env.GW.URL+"/auth/register", map[string]any{
		"email":    email,
		"password": password,
	}, nil)
	mustStatus(t, resp, raw, http.StatusCreated)
}

func login(t *testing.T, env testEnv, email, password string) string {
	t.Helper()
	resp, raw := doJSON(t, env.Client, http.MethodPost, env.GW.URL+"/auth/login", map[string]any{
		"email":    email,
		"password": password,
	}, nil)
	mustStatus(t, resp, raw, http.StatusOK)

	var lr struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(raw, &lr); err != nil {
		t.Fatalf("decode login: %v body=%s", err, string(raw))
	}
	if lr.AccessToken == "" {
		t.Fatalf("empty access_token")
	}
	return lr.AccessToken
}

func createOrder(t *testing.T, env testEnv, token string, items []map[string]any) order.Order {
	t.Helper()
	resp, raw := doJSON(t, env.Client, http.MethodPost, env.GW.URL+"/orders", map[string]any{
		"items": items,
	}, map[string]string{
		"Authorization": "Bearer " + token,
	})
	mustStatus(t, resp, raw, http.StatusCreated)

	var created order.Order
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatalf("decode order: %v body=%s", err, string(raw))
	}
	if created.ID == "" {
		t.Fatalf("empty order id")
	}
	if created.UserID == "" {
		t.Fatalf("empty user_id")
	}
	return created
}

func getOrder(t *testing.T, env testEnv, token, id string) order.Order {
	t.Helper()
	resp, raw := doJSON(t, env.Client, http.MethodGet, env.GW.URL+"/orders/"+id, nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	mustStatus(t, resp, raw, http.StatusOK)

	var got order.Order
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode order: %v body=%s", err, string(raw))
	}
	return got
}

func TestGateway_PublicAPI_Readyz(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	resp, raw := doJSON(t, env.Client, http.MethodGet, env.GW.URL+"/readyz", nil, nil)
	mustStatus(t, resp, raw, http.StatusOK)
}

func TestGateway_PublicAPI_HappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	email := "user@example.com"
	pass := "password123"

	register(t, env, email, pass)
	token := login(t, env, email, pass)

	created := createOrder(t, env, token, []map[string]any{
		{"product_id": "p1", "qty": 2},
		{"product_id": "p2", "qty": 1},
	})

	if created.TotalCents != 11970 {
		t.Fatalf("total_cents=%d", created.TotalCents)
	}

	got := getOrder(t, env, token, created.ID)

	if got.ID != created.ID {
		t.Fatalf("id=%s want=%s", got.ID, created.ID)
	}
	if got.TotalCents != created.TotalCents {
		t.Fatalf("total=%d want=%d", got.TotalCents, created.TotalCents)
	}
}

func TestGateway_PublicAPI_OrdersRequiresAuth(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	resp, raw := doJSON(t, env.Client, http.MethodPost, env.GW.URL+"/orders", map[string]any{
		"items": []map[string]any{{"product_id": "p1", "qty": 1}},
	}, nil)

	mustStatus(t, resp, raw, http.StatusUnauthorized)
}

func TestGateway_PublicAPI_InvalidTokenRejected(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	email := "user2@example.com"
	pass := "password123"

	register(t, env, email, pass)
	tok := login(t, env, email, pass)

	if len(tok) < 5 {
		t.Fatalf("token too short")
	}
	badTok := tok[:len(tok)-1] + "x"

	resp, raw := doJSON(t, env.Client, http.MethodPost, env.GW.URL+"/orders", map[string]any{
		"items": []map[string]any{{"product_id": "p1", "qty": 1}},
	}, map[string]string{
		"Authorization": "Bearer " + badTok,
	})

	mustStatus(t, resp, raw, http.StatusUnauthorized)
}
