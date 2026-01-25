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

func newAuthTS(t *testing.T, jwtSecret string) *httptest.Server {
	t.Helper()

	s := &auth.Server{
		Log:   zap.NewNop(),
		Store: auth.NewStore(),
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

	s := &catalog.Server{Store: catalog.NewStore()}

	h := catalog.NewHandler(s, catalog.HTTPDeps{
		Log:     zap.NewNop(),
		Service: "catalog",
	})

	return httptest.NewServer(h)
}

func newOrderTS(t *testing.T, catalogURL string) *httptest.Server {
	t.Helper()

	s := &order.Server{
		Store:   order.NewStore(),
		Catalog: order.NewCatalogClient(catalogURL),
	}

	h := order.NewHandler(s, order.HTTPDeps{
		Log:     zap.NewNop(),
		Service: "order",
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
			// Registry: nil
		},
	)
	if err != nil {
		t.Fatalf("gateway.NewHandler: %v", err)
	}

	return httptest.NewServer(h)
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

func TestGateway_PublicAPI_HappyPath(t *testing.T) {
	const jwtSecret = "test-secret"

	authTS := newAuthTS(t, jwtSecret)
	t.Cleanup(authTS.Close)

	catalogTS := newCatalogTS(t)
	t.Cleanup(catalogTS.Close)

	orderTS := newOrderTS(t, catalogTS.URL)
	t.Cleanup(orderTS.Close)

	gwTS := newGatewayTS(t, jwtSecret, authTS.URL, catalogTS.URL, orderTS.URL)
	t.Cleanup(gwTS.Close)

	c := &http.Client{}

	{
		resp, _ := doJSON(t, c, http.MethodPost, gwTS.URL+"/auth/register", map[string]any{
			"email":    "user@example.com",
			"password": "password123",
		}, nil)

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("register status=%d", resp.StatusCode)
		}
	}

	var accessToken string
	{
		resp, raw := doJSON(t, c, http.MethodPost, gwTS.URL+"/auth/login", map[string]any{
			"email":    "user@example.com",
			"password": "password123",
		}, nil)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login status=%d body=%s", resp.StatusCode, string(raw))
		}

		var lr struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(raw, &lr); err != nil {
			t.Fatalf("decode login: %v body=%s", err, string(raw))
		}
		if lr.AccessToken == "" {
			t.Fatalf("empty access_token")
		}
		accessToken = lr.AccessToken
	}

	var created order.Order
	{
		resp, raw := doJSON(t, c, http.MethodPost, gwTS.URL+"/orders", map[string]any{
			"items": []map[string]any{
				{"product_id": "p1", "qty": 2},
				{"product_id": "p2", "qty": 1},
			},
		}, map[string]string{
			"Authorization": "Bearer " + accessToken,
		})

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create order status=%d body=%s", resp.StatusCode, string(raw))
		}

		if err := json.Unmarshal(raw, &created); err != nil {
			t.Fatalf("decode order: %v body=%s", err, string(raw))
		}

		if created.TotalCents != 11970 {
			t.Fatalf("total_cents=%d", created.TotalCents)
		}
		if created.ID == "" {
			t.Fatalf("empty order id")
		}
		if created.UserID == "" {
			t.Fatalf("empty user_id")
		}
	}

	{
		resp, raw := doJSON(t, c, http.MethodGet, gwTS.URL+"/orders/"+created.ID, nil, map[string]string{
			"Authorization": "Bearer " + accessToken,
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get order status=%d body=%s", resp.StatusCode, string(raw))
		}

		var got order.Order
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode order: %v body=%s", err, string(raw))
		}
		if got.ID != created.ID {
			t.Fatalf("id=%s want=%s", got.ID, created.ID)
		}
		if got.TotalCents != created.TotalCents {
			t.Fatalf("total=%d want=%d", got.TotalCents, created.TotalCents)
		}
	}
}

func TestGateway_PublicAPI_OrdersRequiresAuth(t *testing.T) {
	const jwtSecret = "test-secret"

	authTS := newAuthTS(t, jwtSecret)
	t.Cleanup(authTS.Close)

	catalogTS := newCatalogTS(t)
	t.Cleanup(catalogTS.Close)

	orderTS := newOrderTS(t, catalogTS.URL)
	t.Cleanup(orderTS.Close)

	gwTS := newGatewayTS(t, jwtSecret, authTS.URL, catalogTS.URL, orderTS.URL)
	t.Cleanup(gwTS.Close)

	c := &http.Client{}

	resp, raw := doJSON(t, c, http.MethodPost, gwTS.URL+"/orders", map[string]any{
		"items": []map[string]any{{"product_id": "p1", "qty": 1}},
	}, nil)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(raw))
	}
}
