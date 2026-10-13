# Finance Services

Учебный репозиторий с двумя Go-сервисами:

- `gateway` — HTTP-шлюз;
- `ledger` — сервис бизнес-логики для хранения финансовых транзакций в памяти.

## Структура

```text
.
├── gateway/
│   ├── go.mod
│   └── main.go
└── ledger/
    ├── go.mod
    ├── main.go
    └── transaction.go
```

## Требования

- Go 1.23+;
- Git;
- curl или браузер для проверки Gateway.

## Запуск Gateway

```bash
cd gateway
go run .
```

Сервис запускается на порту `8080`.

Проверка:

```bash
curl -i http://localhost:8080/ping
```

Ожидаемый ответ:

```text
pong
```

## Запуск Ledger

```bash
cd ledger
go run .
```

Ledger добавит несколько тестовых транзакций в память и выведет их в консоль.

## Проверка кода

Проверить сборку обоих модулей:

```bash
cd gateway
go build .

cd ../ledger
go build .
```