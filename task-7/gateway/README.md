# Gateway

HTTP-шлюз финансового приложения.

Gateway принимает запросы на порту 8080 и передаёт их Ledger.
Сам Gateway не подключается к PostgreSQL или Redis и не хранит данные.

## Архитектура

Клиент → Gateway :8080 → Ledger :8081 → PostgreSQL / Redis.

Публичные пути /api/... сохранены.
Gateway передаёт статус и тело ответа Ledger.

Если соединение с Ledger не удалось, возвращается 502:
{"error":"ledger service unavailable"}.

## Порядок запуска

1. Из корня репозитория запустить инфраструктуру:

```powershell
docker compose up -d
```

2. Применить миграции и запустить Ledger по инструкции
   в ../ledger/README.md.
3. Оставить Ledger работающим в отдельном терминале.
4. Запустить Gateway в другом терминале.

## Запуск Gateway

Из корня репозитория:

```powershell
cd gateway

$env:LEDGER_URL = "http://127.0.0.1:8081"

go run .
```

Адрес Gateway: http://localhost:8080.

LEDGER_URL по умолчанию: http://127.0.0.1:8081.
Указывайте только HTTP(S)-адрес сервиса, без пути /api,
параметров запроса и учётных данных.

## Проверка API

Команды выполняются в третьем PowerShell,
пока Gateway и Ledger работают.

### Создание бюджета

```powershell
$body = @{
    category = "demo"
    limit = 5000
} | ConvertTo-Json

$result = Invoke-WebRequest `
    -Uri "http://localhost:8080/api/budgets" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing

$result.StatusCode
$result.Content
```

Ожидается 201.
Повторная запись той же категории обновляет её лимит.

### Расход в пределах бюджета

```powershell
$body = @{
    amount = 1000
    category = "demo"
    description = "demo purchase"
    date = "2026-11-02"
} | ConvertTo-Json

$result = Invoke-WebRequest `
    -Uri "http://localhost:8080/api/transactions" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing

$result.StatusCode
$result.Content
```

Ожидается 201 при наличии достаточного остатка бюджета.

### Превышение бюджета

```powershell
$body = @{
    amount = 6000
    category = "demo"
    description = "must be rejected"
    date = "2026-11-02"
} | ConvertTo-Json

try {
    $result = Invoke-WebRequest `
        -Uri "http://localhost:8080/api/transactions" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body `
        -UseBasicParsing `
        -ErrorAction Stop

    $result.StatusCode
    $result.Content
}
catch {
    if ($null -ne $_.Exception.Response) {
        [int]$_.Exception.Response.StatusCode
    }
    $_.ErrorDetails.Message
}
```

Ожидается 409 и budget exceeded.
Отклонённый расход не должен появиться в PostgreSQL.

Примеры меняют рабочие данные.
Повторный запуск успешного POST добавляет новый расход.

### Чтение списков через cURL

В Windows PowerShell используйте curl.exe:

```powershell
curl.exe -i "http://localhost:8080/api/budgets"
curl.exe -i "http://localhost:8080/api/transactions"
```

Ожидается 200.

### Проверка отчёта и кэша

Выполните команды подряд:

```powershell
curl.exe -i "http://localhost:8080/api/reports/summary?from=2026-11-01&to=2026-11-30"
curl.exe -i "http://localhost:8080/api/reports/summary?from=2026-11-01&to=2026-11-30"
```

Ожидается 200.

В логах Ledger второй запрос должен показать HIT.
Первый запрос также может быть HIT, если ключ уже существовал.

Расчёт отчёта и работа с Redis выполняются только в Ledger.

### Сохранность данных

1. Создайте бюджет и расход через Gateway.
2. Остановите только Ledger через Ctrl+C.
3. Снова запустите Ledger с тем же DATABASE_URL.
4. Повторите GET бюджетов и транзакций через Gateway.

Ранее сохранённые данные должны остаться доступны.

## Автоматические тесты

Из папки gateway:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\test.ps1
```

Скрипт выполняет:
- go test ./... -count=1;
- go vet ./...;
- go build .

Тесты проверяют передачу методов, путей, параметров запроса,
JSON-тела, Content-Type и HTTP-статусов.

Отдельно проверяются некорректный LEDGER_URL
и ответ 502 при сбое соединения.

Реальные Ledger, PostgreSQL и Redis для тестов Gateway не нужны.