# Ledger

Сервис бюджетов, расходов и отчётов.

Ledger:
- хранит бюджеты и расходы в PostgreSQL;
- проверяет входные данные;
- запрещает добавление расхода сверх бюджета категории;
- формирует отчёты за период;
- кэширует отчёты в Redis на 30 секунд.

Адрес сервиса: http://127.0.0.1:8081.

## Архитектура

Клиент → Gateway :8080 → Ledger :8081 → PostgreSQL / Redis.

HTTP-обработчики находятся в internal/api.
Бизнес-логика, подключение к БД и кэш — в internal/ledger.
SQL-миграции — в migrations.

## Подготовка инфраструктуры

Команды ниже выполняются в PowerShell из корня репозитория,
где находится compose.yaml:

```powershell
docker compose up -d
docker compose ps
docker compose port postgres 5432
docker compose port redis 6379
```

В текущей конфигурации:
- PostgreSQL: 127.0.0.1:5444;
- Redis: 127.0.0.1:6379.

При первом запуске с пустым томом контейнер PostgreSQL создаёт
базу cashapp согласно POSTGRES_DB в compose.yaml.

Данные PostgreSQL хранятся в именованном Docker-томе.
Не используйте docker compose down -v, если данные нужно сохранить.

## Миграции Goose

Установка:

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Если goose не найден, добавьте каталог Go-инструментов в PATH
текущего терминала:

```powershell
$goBin = go env GOBIN
if (-not $goBin) {
    $goBin = Join-Path (go env GOPATH) "bin"
}
$env:Path = "$goBin;$env:Path"
```

Из корня репозитория:

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp?sslmode=disable"

goose -dir ./ledger/migrations postgres "$env:DATABASE_URL" up
goose -dir ./ledger/migrations postgres "$env:DATABASE_URL" status
```

Откат последней миграции:

```powershell
goose -dir ./ledger/migrations postgres "$env:DATABASE_URL" down
```

Внимание: откат первой миграции удаляет таблицы budgets и expenses
вместе с данными. Не выполняйте откат при обычном запуске.

## Запуск Ledger

После успешного применения миграций:

```powershell
cd ledger

$env:DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp?sslmode=disable"
$env:REDIS_ADDR = "127.0.0.1:6379"
$env:REDIS_DB = "0"
$env:REDIS_PASSWORD = ""

go run .
```

Ledger должен оставаться запущенным в отдельном терминале.

### Настройки PostgreSQL

DATABASE_URL имеет приоритет.

Если DATABASE_URL отсутствует, используются:

| Переменная | Значение по умолчанию |
|---|---|
| DB_HOST | localhost |
| DB_PORT | 5432 |
| DB_USER | postgres |
| DB_PASS | postgres |
| DB_NAME | cashapp |

Для текущего Docker-проброса без DATABASE_URL необходимо задать
DB_PORT=5444.

sslmode=disable используется для локальной разработки.

### Настройки Redis

| Переменная | Значение по умолчанию |
|---|---|
| REDIS_ADDR | localhost:6379 |
| REDIS_DB | 0 |
| REDIS_PASSWORD | без пароля |

Если PostgreSQL или Redis недоступны при запуске, Ledger завершается
с сообщением об ошибке.

Если Redis становится недоступен во время получения отчёта,
сервис пытается рассчитать отчёт через PostgreSQL.

## API

| Метод | Путь | Назначение |
|---|---|---|
| POST | /api/budgets | Создать или обновить бюджет |
| GET | /api/budgets | Получить бюджеты |
| POST | /api/transactions | Добавить расход |
| GET | /api/transactions | Получить расходы |
| GET | /api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD | Получить отчёт |

Бюджет сохраняется через UPSERT по уникальной категории.

При превышении бюджета возвращается 409:
{"error":"budget exceeded"}.

Если бюджет категории отсутствует, расход разрешён без лимита.

Лимит проверяется по всем расходам категории, а не по календарному месяцу.
Поле period="month" сохранено для совместимости HTTP-ответов.

## Отчёт и кэш

Обе граничные даты входят в отчёт.

Ключ Redis:
report:summary:<from>:<to>

TTL: 30 секунд.

В логах:
- MISS — ключ отсутствует;
- SET — отчёт записан в кэш;
- HIT — ответ получен из кэша.

После добавления расхода отчёт может оставаться устаревшим
до истечения TTL.

## Автоматические тесты

Тесты используют отдельные базы:
- cashapp_ledger_test;
- cashapp_api_test.

И Redis DB 15, зарезервированную для тестов.

Создайте базы один раз из корня репозитория:

```powershell
docker compose exec postgres psql -U postgres -d postgres -c "CREATE DATABASE cashapp_ledger_test TEMPLATE template0;"
docker compose exec postgres psql -U postgres -d postgres -c "CREATE DATABASE cashapp_api_test TEMPLATE template0;"
```

Если база уже существует, повторно создавать её не нужно.

Примените миграции:

```powershell
$env:LEDGER_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_ledger_test?sslmode=disable"
$env:API_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_api_test?sslmode=disable"

goose -dir ./ledger/migrations postgres "$env:LEDGER_TEST_DATABASE_URL" up
goose -dir ./ledger/migrations postgres "$env:API_TEST_DATABASE_URL" up
```

Запустите из папки ledger:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\test.ps1
```

Скрипт запускает тесты и go vet.

Тестовые таблицы очищаются перед сценариями и после них.
Рабочая cashapp для тестов не используется.
Не запускайте одновременно несколько копий тестов с теми же базами.

Тест кэша проверяет запись, TTL, попадание в кэш и пересчёт
после удаления тестового ключа. Естественного истечения
30 секунд он не ожидает.

## Ограничения учебной реализации

- Проверка лимита и INSERT выполняются отдельными запросами:
  одновременные расходы могут обойти проверку.
- В Go суммы представлены float64, в PostgreSQL — NUMERIC(14,2).
  Для точных денежных расчётов требуется отдельное улучшение
  представления сумм и правил округления.
- Нет авторизации и разделения данных по пользователям.
- Импорт LoadBudgets не является атомарным.