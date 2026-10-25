# Gateway

HTTP-фасад над сервисом `ledger`. Все ответы — `Content-Type: application/json; charset=utf-8`.
Ошибки возвращаются в едином формате: `{"error":"..."}`.

## Эндпоинты

| Метод | Путь               | Описание                            |
|-------|--------------------|-------------------------------------|
| GET   | `/ping`            | health-check (`pong`)               |
| POST  | `/api/transactions`| создать транзакцию                  |
| GET   | `/api/transactions`| список транзакций (или `[]`)        |
| POST  | `/api/budgets`     | создать/обновить бюджет             |
| GET   | `/api/budgets`     | список бюджетов (или `[]`)          |

## Коды ответов

- `201 Created` — успешное создание;
- `200 OK` — успешный список;
- `400 Bad Request` — ошибка валидации/JSON;
- `409 Conflict` — `{"error":"budget exceeded"}`;
- `500 Internal Server Error` — прочие ошибки.

## Примеры cURL

Создать транзакцию:

```bash
curl -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'
```

Список транзакций:

```bash
curl http://localhost:8080/api/transactions
```

Создать бюджет:

```bash
curl -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" \
  -d '{"category":"еда","limit":20000}'
```

Список бюджетов:

```bash
curl http://localhost:8080/api/budgets
```

Проверить превышение бюджета (при лимите 1000 на «еда»):

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":5000,"category":"еда","description":"ужин","date":"2025-09-11"}'
# HTTP/1.1 409 Conflict
# {"error":"budget exceeded"}
```