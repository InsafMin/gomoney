# Analytics Service

Сервис аналитики и отчетности по финансовым данным.

## Описание

Формирует отчеты, статистику по категориям и тренды. Потребляет события транзакций из Kafka.

## Запуск

```bash
make run          # Локально
make test         # Тесты
```

## API примеры

```bash
TOKEN="your_token"

# Месячный отчет
curl "http://localhost:8080/api/v1/analytics/monthly/2024/12" \
  -H "Authorization: Bearer $TOKEN"

# Статистика по категориям
curl "http://localhost:8080/api/v1/analytics/category-stats?type=expense" \
  -H "Authorization: Bearer $TOKEN"
```
