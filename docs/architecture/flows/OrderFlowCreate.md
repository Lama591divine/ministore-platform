```mermaid
sequenceDiagram
  autonumber
  participant C as Client
  participant GW as Gateway
  participant O as Order
  participant CAT as Catalog
  participant DB as Postgres (order)

  C->>GW: POST /orders (Authorization: Bearer JWT) {items}
  GW->>GW: AuthJWT validate token (only for /orders/*)
  alt missing/invalid token
    GW-->>C: 401
  else token ok
    GW->>O: proxy POST /orders (Authorization header preserved)
    O->>O: AuthJWT validate token
    alt missing/invalid token
      O-->>GW: 401
      GW-->>C: 401
    else token ok
      loop for each item
        O->>CAT: GET /products/{product_id}
        alt 404
          CAT-->>O: 404
          O-->>GW: 400 invalid product_id
          GW-->>C: 400
        else non-200
          CAT-->>O: 5xx/4xx
          O-->>GW: 502 catalog error
          GW-->>C: 502
        else timeout/unavailable
          O-->>GW: 503 catalog unavailable
          GW-->>C: 503
        else 200
          CAT-->>O: 200 {price_cents,...}
          O->>O: accumulate total (overflow check)
        end
      end
      O->>DB: BEGIN; INSERT orders; INSERT order_items; COMMIT
      DB-->>O: OK
      O-->>GW: 201 {order}
      GW-->>C: 201 {order}
    end
  end
```