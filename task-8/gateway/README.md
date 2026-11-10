# Gateway

HTTP-приложение для работы с бюджетами, расходами и отчётами.

Gateway принимает запросы на порту 8080. Обработчики вызывают
публичный интерфейс Ledger внутри того же процесса.

Отдельный HTTP-сервис Ledger не запускается.
Переменная LEDGER_URL не используется.

## Архитектура

Клиент → Gateway :8080 → ledger/service.Service
→ бизнес-логика Ledger → PostgreSQL / Redis.

Основные части:
- main.go — создание приложения и запуск HTTP-сервера;
- internal/api — HTTP-маршруты, обработчики и преобразование данных;
- ledger/app — фабрика сервиса и управление подключениями;
- ledger/service — публичный интерфейс Ledger.

Gateway получает сервис через ledger/app.New(ctx).
HTTP-обработчики работают с ledger/service.Service.

## Подготовка инфраструктуры

Команды выполняются из корня task-8, где находится compose.yaml:

```powershell
docker compose up -d
docker compose ps
```

В текущей локальной конфигурации:
- PostgreSQL: 127.0.0.1:5444;
- Redis: 127.0.0.1:6379;
- рабочая база: cashapp.

Перед первым запуском примените миграции по инструкции
в ../ledger/README.md.

Не используйте docker compose down -v, если данные нужно сохранить.

## Запуск

Из корня task-8:

```powershell
cd gateway

$env:GOWORK = "off"

$env:DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp?sslmode=disable"

$env:REDIS_ADDR = "127.0.0.1:6379"
$env:REDIS_DB = "0"
$env:REDIS_PASSWORD = ""

go run .
```

Адрес HTTP-приложения: http://localhost:8080.

При запуске фабрика проверяет PostgreSQL и Redis.
Если подключиться не удалось, HTTP-сервер не запускается.

Для остановки нажмите Ctrl+C. Приложение начинает завершение
HTTP-сервера с тайм-аутом 10 секунд, затем закрывает ресурсы Ledger.

Настройки подключений описаны в ../ledger/README.md.

## API

| Метод | Путь | Назначение |
|---|---|---|
| POST | /api/budgets | Создать или обновить бюджет |
| GET | /api/budgets | Получить бюджеты |
| POST | /api/transactions | Добавить расход |
| GET | /api/transactions | Получить расходы |
| GET | /api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD | Получить отчёт |

## Ручная проверка

Команды выполняются в другом PowerShell, пока Gateway работает.
Примеры изменяют рабочие данные.

### Создать бюджет

Используйте новую категорию для каждого полного запуска примера:

```powershell
$category = "demo-" + [guid]::NewGuid().ToString("N")

$body = @{
    category = $category
    limit = 5000
} | ConvertTo-Json

Invoke-WebRequest `
    -Uri "http://localhost:8080/api/budgets" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing
```

Ожидается HTTP 201.
Повторная запись бюджета той же категории обновляет лимит.

### Добавить расход

В том же терминале:

```powershell
$body = @{
    amount = 1000
    category = $category
    description = "demo purchase"
    date = "2026-11-02"
} | ConvertTo-Json

Invoke-WebRequest `
    -Uri "http://localhost:8080/api/transactions" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing
```

Ожидается HTTP 201.
Повторный успешный POST добавляет ещё один расход.

### Проверить превышение бюджета

```powershell
$body = @{
    amount = 6000
    category = $category
    description = "must be rejected"
    date = "2026-11-02"
} | ConvertTo-Json

try {
    Invoke-WebRequest `
        -Uri "http://localhost:8080/api/transactions" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body `
        -UseBasicParsing `
        -ErrorAction Stop
}
catch {
    if ($null -ne $_.Exception.Response) {
        [int]$_.Exception.Response.StatusCode
    }
    $_.ErrorDetails.Message
}
```

Ожидается HTTP 409 и ошибка budget exceeded.
Отклонённый расход не должен сохраняться.

### Получить списки и отчёт

В Windows PowerShell используйте curl.exe:

```powershell
curl.exe -i "http://localhost:8080/api/budgets"
curl.exe -i "http://localhost:8080/api/transactions"

curl.exe -i "http://localhost:8080/api/reports/summary?from=2026-11-01&to=2026-11-30"
curl.exe -i "http://localhost:8080/api/reports/summary?from=2026-11-01&to=2026-11-30"
```

Ожидается HTTP 200.

Повторный запрос отчёта в пределах TTL может использовать Redis-кэш.
Если ключ уже существовал, первый запрос также может попасть в кэш.

### Проверить сохранность данных

1. Создайте бюджет и расход.
2. Остановите Gateway через Ctrl+C.
3. Запустите Gateway с тем же DATABASE_URL.
4. Повторите GET бюджетов и транзакций.

Сохранённые данные должны остаться доступны.

## Обычные автоматические проверки

Из папки gateway:

```powershell
.\test.ps1
```

Скрипт выполняет:

```text
go test ./... -count=1
go vet ./...
go build .
```

Интеграционный пакет в этот режим не включается.

## Интеграционные HTTP-тесты

Тесты находятся в internal/api и включаются тегом integration.
Они используют отдельную базу cashapp_api_test.

Подготовка тестовых баз и миграций описана в ../ledger/README.md.

Из папки gateway:

```powershell
$env:API_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_api_test?sslmode=disable"

$env:TEST_REDIS_ADDR = "127.0.0.1:6379"
$env:TEST_REDIS_PASSWORD = ""

.\test.ps1 -Integration
```

Помимо обычных проверок скрипт выполняет:

```text
go vet -tags=integration ./internal/api
go test -tags=integration ./internal/api -count=1 -parallel=1
```

Проверки используют инфраструктуру тестового окружения.
Отдельно запускать Gateway через go run для них не нужно.

Не запускайте одновременно несколько копий тестов,
использующих ту же тестовую базу или общие ключи Redis.