# Transaction Service

Сервис управления финансовыми транзакциями и категориями.

## Описание

Управляет транзакциями, категориями, балансом. Публикует события в Kafka для аналитики.

## Запуск

```bash
make run          # Локально
make test         # Тесты
```

## API примеры

```bash
TOKEN="your_token"

# Создать транзакцию
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000,"currency":"RUB","type":"income","description":"Зарплата"}'

# Получить баланс
curl "http://localhost:8080/api/v1/transactions/balance" \
  -H "Authorization: Bearer $TOKEN"
```
