# Gateway и Ledger — тестирование

## Запуск тестов

Команды выполняются из папки gateway.

Все тесты:

```powershell
go test ./...
```

Подробный запуск без кеша:

```powershell
go test -v -count=1 ./...
```

## Покрытие

```powershell
go test ./... -count=1 -cover -coverprofile=cover.out
go tool cover -html=cover.out -o cover.html
go tool cover -func=cover.out
```

На момент проверки:
- internal/ledger — 78.9%;
- internal/api — 82.5%.

## Что проверяется

- Валидация транзакций и бюджетов.
- Добавление транзакций в пределах бюджета.
- Отказ при суммарном превышении бюджета.
- Отсутствие сохранения отклонённых транзакций.
- HTTP-эндпоинты создания и получения бюджетов и транзакций.
- Коды ответов 201, 200, 400 и 409.
- JSON-ответы и заголовок Content-Type.
- Изоляция сценариев через Reset().