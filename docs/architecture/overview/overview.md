```mermaid
    flowchart LR
    Client[Client / Frontend] -->|HTTP| GW[Gateway :8080]
    
    GW -->|/auth/* proxy| AUTH[Auth :8081]
    GW -->|/products/* proxy| CAT[Catalog :8082]
    GW -->|/orders/* proxy| ORD[Order :8083]
    
    ORD -->|HTTP GET /products/{id}| CAT
    
    AUTH --> PG1[(Postgres)]
    CAT --> PG2[(Postgres)]
    ORD --> PG3[(Postgres)]
    
    subgraph Observability
    PROM[Prometheus]
    end
    
    PROM -. scrape /metrics (token) .-> GW
    PROM -. scrape /metrics (token) .-> AUTH
    PROM -. scrape /metrics (token) .-> CAT
    PROM -. scrape /metrics (token) .-> ORD
```