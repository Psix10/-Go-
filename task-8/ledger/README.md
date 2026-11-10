# Ledger

Go-модуль бизнес-логики бюджетов, расходов и отчётов.

Ledger используется Gateway как библиотека внутри одного процесса.
Собственного HTTP-сервера и main.go в модуле нет.
Команда go run . из папки ledger не используется.

## Структура

- service — публичный интерфейс Service, типы данных и ошибки;
- app — фабрика приложения и настройки подключений;
- internal/domain — доменные типы и валидация;
- internal/service — бизнес-логика и её тесты;
- internal/repository/pg — PostgreSQL-репозитории;
- internal/repository/redis — Redis-кэш отчётов;
- internal/testutil — вспомогательный код тестов;
- integration — интеграционные тесты PostgreSQL и Redis;
- migrations — SQL-миграции Goose.

HTTP-обработчики находятся в gateway/internal/api.

## Публичный интерфейс

ledger/service.Service предоставляет методы:

- SetBudget;
- ListBudgets;
- AddTransaction;
- ListTransactions;
- GetReportSummary.

Все методы принимают context.Context.

Пакет service также экспортирует типы Budget, Transaction,
CategorySummary, ReportSummary, ValidationError и ошибку
ErrBudgetExceeded.

## Создание приложения

Фабрика:

```go
app.New(ctx)
```

возвращает:
- service.Service;
- функцию закрытия ресурсов func() error;
- ошибку инициализации.

Фабрика:
1. Открывает подключение к PostgreSQL и выполняет Ping.
2. Создаёт Redis-клиент и выполняет Ping.
3. Создаёт репозитории и кэш.
4. Собирает бизнес-сервис.

При ошибке подключения уже открытые ресурсы закрываются.
Вызывающий код должен вызвать возвращённую функцию закрытия
после завершения работы с сервисом.

Gateway делает это в своей точке входа.

## Подготовка инфраструктуры

Команды выполняются из корня task-8:

```powershell
docker compose up -d
docker compose ps
docker compose port postgres 5432
docker compose port redis 6379
```

В текущей конфигурации:
- PostgreSQL: 127.0.0.1:5444;
- Redis: 127.0.0.1:6379;
- рабочая база: cashapp.

Не используйте docker compose down -v, если данные нужно сохранить.

## Миграции Goose

Если Goose ещё не установлен:

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Если команда goose не найдена, добавьте каталог Go-инструментов
в PATH текущего терминала:

```powershell
$goBin = go env GOBIN
if (-not $goBin) {
    $goBin = Join-Path (go env GOPATH) "bin"
}
$env:Path = "$goBin;$env:Path"
```

Применение миграций из корня task-8:

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp?sslmode=disable"

goose -dir ./ledger/migrations postgres "$env:DATABASE_URL" up
goose -dir ./ledger/migrations postgres "$env:DATABASE_URL" status
```

Миграции создают таблицы budgets и expenses,
а также индекс расходов по категории и дате.

Не выполняйте down при обычном запуске:
откат первой миграции удаляет таблицы вместе с данными.

## Настройки PostgreSQL

Непустой DATABASE_URL имеет приоритет над отдельными настройками.

Если он не задан, используются:

| Переменная | Значение по умолчанию |
|---|---|
| DB_HOST | localhost |
| DB_PORT | 5432 |
| DB_USER | postgres |
| DB_PASS | postgres |
| DB_NAME | cashapp |

При построении адреса из отдельных настроек используется
sslmode=disable.

Для текущего Docker-проброса задайте DB_PORT=5444
либо полный DATABASE_URL.

Локальный пример:

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp?sslmode=disable"
```

Параметры пула PostgreSQL:
- максимум открытых подключений — 10;
- максимум простаивающих подключений — 5;
- максимальное время жизни подключения — 30 минут;
- тайм-аут первоначального Ping — 5 секунд.

## Настройки Redis

| Переменная | Значение по умолчанию |
|---|---|
| REDIS_ADDR | localhost:6379 |
| REDIS_DB | 0 |
| REDIS_PASSWORD | без пароля |

REDIS_DB должен быть неотрицательным целым числом.

Тайм-ауты клиента:
- подключение — 5 секунд;
- чтение — 3 секунды;
- запись — 3 секунды;
- первоначальный Ping — 5 секунд.

И PostgreSQL, и Redis должны быть доступны при создании приложения.

## Отчёт и кэш

Интеграционный тест отчёта проверяет:
- включение обеих граничных дат периода;
- исключение расходов за пределами периода;
- запись отчёта в Redis;
- TTL до 30 секунд;
- получение ранее рассчитанного отчёта из кэша;
- пересчёт после удаления тестового ключа.

Формат ключа:

```text
report:summary:<from>:<to>
```

TTL отчёта — 30 секунд.

В логах используются обозначения MISS, SET и HIT.

Проверенный сценарий сохраняет закэшированный отчёт после
добавления нового расхода. Поэтому отчёт может отражать данные
на момент предыдущего расчёта до истечения TTL.

Тест не ожидает естественного истечения 30 секунд:
для проверки пересчёта он удаляет свой ключ.

## Обычные автоматические проверки

Из папки ledger:

```powershell
.\test.ps1
```

Скрипт выполняет:

```text
go test ./... -count=1
go vet ./...
go build ./...
```

Интеграционные тесты включаются отдельно тегом integration.

## Подготовка интеграционных тестов

Используются отдельные базы:
- cashapp_ledger_test — интеграционные тесты Ledger;
- cashapp_api_test — интеграционные HTTP-тесты Gateway.

Redis DB 15 зарезервирована для тестов.

Создайте базы один раз из корня task-8:

```powershell
docker compose exec postgres psql -U postgres -d postgres -c "CREATE DATABASE cashapp_ledger_test TEMPLATE template0;"
docker compose exec postgres psql -U postgres -d postgres -c "CREATE DATABASE cashapp_api_test TEMPLATE template0;"
```

Если база уже существует, повторно создавать её не нужно.

Примените миграции к обеим тестовым базам:

```powershell
$env:LEDGER_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_ledger_test?sslmode=disable"

$env:API_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_api_test?sslmode=disable"

goose -dir ./ledger/migrations postgres "$env:LEDGER_TEST_DATABASE_URL" up
goose -dir ./ledger/migrations postgres "$env:API_TEST_DATABASE_URL" up
```

Тесты очищают тестовые данные.
Не указывайте рабочую базу в тестовых переменных.

Не запускайте одновременно несколько копий проверок,
использующих те же базы или общие ключи Redis.

## Запуск интеграционных тестов Ledger

Из папки ledger:

```powershell
$env:LEDGER_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_ledger_test?sslmode=disable"

$env:TEST_REDIS_ADDR = "127.0.0.1:6379"
$env:TEST_REDIS_PASSWORD = ""

.\test.ps1 -Integration
```

Помимо обычных проверок скрипт выполняет:

```text
go vet -tags=integration ./integration
go test -tags=integration ./integration -count=1 -parallel=1
```

Интеграционные тесты вызывают сервис через фабрику app.New
и используют настоящие PostgreSQL и Redis.

Проверяются бюджеты, UPSERT, списки, граничные значения лимита,
суммарные расходы, расходы без бюджета и кэш отчёта.

Интеграционные HTTP-тесты запускаются отдельным скриптом Gateway.
Команды приведены в ../gateway/README.md.

## Ограничения

Суммы и правила конкурентного изменения бюджета требуют
отдельной оценки перед применением в реальной финансовой системе.

Успешный прогон текущих тестов не доказывает корректность
всех возможных конкурентных сценариев и отказов инфраструктуры.