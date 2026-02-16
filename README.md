# MiniStore

Небольшой учебный проект с микросервисами: `gateway`, `auth`, `catalog`, `order`.

- **gateway** — единая точка входа (reverse proxy), health/ready/metrics
- **auth** — регистрация/логин, выпуск JWT
- **catalog** — чтение каталога товаров
- **order** — создание/получение заказов, проверка JWT, расчёт total через catalog

---

## Architecture

- Overview diagram: `docs/architecture/overview.md`
- Request flows (sequence diagrams): `docs/architecture/flows.md`

---

## Services

### Gateway (`gateway`, :8080)
- Routes:
    - `/auth/*` -> `auth`
    - `/products/*` -> `catalog`
    - `/orders/*` -> `order` (JWT check on gateway)
- Infra:
    - `GET /healthz`
    - `GET /readyz` (checks auth/catalog/order `/readyz`)
    - `GET /metrics` (token-protected)

### Auth (`auth`, :8081)
- API:
    - `POST /auth/register`
    - `POST /auth/login` -> `{ "access_token": "..." }`
    - `GET /auth/whoami`
- Infra:
    - `GET /healthz`
    - `GET /readyz` (DB ping)
    - `GET /metrics` (token-protected)

### Catalog (`catalog`, :8082)
- API:
    - `GET /products`
    - `GET /products/{id}`
- Infra:
    - `GET /healthz`
    - `GET /readyz` (store ping)
    - `GET /metrics` (token-protected)

### Order (`order`, :8083)
- API (JWT required):
    - `POST /orders`
    - `GET /orders/{id}`
- Notes:
    - On create, fetches product price from `catalog` to compute `total_cents`
    - Access control: users can only read their own orders
- Infra:
    - `GET /healthz`
    - `GET /readyz` (store ping)
    - `GET /metrics` (token-protected)

---

## Configuration (env)

Common:
- `JWT_SECRET` — required (min 32 chars)
- `METRICS_TOKEN` — token for `/metrics`

Gateway:
- `PORT` (default `8080`)
- `AUTH_URL` (default `http://auth:8081`)
- `CATALOG_URL` (default `http://catalog:8082`)
- `ORDER_URL` (default `http://order:8083`)

Auth:
- `PORT` (default `8081`)
- `POSTGRES_DSN` — required

Catalog:
- `PORT` (default `8082`)
- `POSTGRES_DSN` — required (or `ALLOW_MEMSTORE=1` for dev)

Order:
- `PORT` (default `8083`)
- `CATALOG_URL` (default `http://catalog:8082`)
- `POSTGRES_DSN` — required (or `ALLOW_MEMSTORE=1` for dev)
