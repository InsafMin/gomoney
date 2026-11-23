# API Gateway

Единая точка входа для всех клиентских запросов.

## Описание

REST API → gRPC транслятор с аутентификацией, rate limiting и circuit breaker.

## Запуск

```bash
make run          # Локально (порт 8080)
```

## API (см. openapi.yml)

- `/api/v1/auth/*` - Аутентификация
- `/api/v1/transactions/*` - Транзакции
- `/api/v1/categories/*` - Категории  
- `/api/v1/analytics/*` - Аналитика
- `/health` - Health check

## Быстрый тест

```bash
# Регистрация
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass123"}'

# Вход и получение токена
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass123"}'
```
