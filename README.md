# MiniStore

MiniStore — учебный проект на Go: микросервисы **auth**, **catalog**, **order** за **API Gateway**.  

## Архитектура

- **gateway** (порт `8080`) — единая точка входа, маршрутизация, проверка JWT, прокидывание `X-User-Id`
- **auth** (порт `8081`) — регистрация/логин, выдача JWT
- **catalog** (порт `8082`) — список товаров и цены (`price_cents`)
- **order** (порт `8083`) — создание/получение заказа, расчёт `total_cents` через запросы в catalog

Схема:
Client → gateway → (auth | catalog | order)