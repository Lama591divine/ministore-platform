```mermaid
sequenceDiagram
  autonumber
  participant C as Client
  participant GW as Gateway
  participant O as Order
  participant DB as Postgres (order)

  C->>GW: GET /orders/{id} (Authorization: Bearer JWT)
  GW->>GW: AuthJWT validate token
  alt invalid token
    GW-->>C: 401
  else ok
    GW->>O: proxy GET /orders/{id}
    O->>O: AuthJWT validate token
    O->>DB: SELECT order; SELECT items
    DB-->>O: order + items / no rows
    alt not found
      O-->>GW: 404
      GW-->>C: 404
    else found
      O->>O: if order.user_id != token.user_id -> 403
      alt чужой заказ
        O-->>GW: 403
        GW-->>C: 403
      else свой заказ
        O-->>GW: 200 {order}
        GW-->>C: 200 {order}
      end
    end
  end
```