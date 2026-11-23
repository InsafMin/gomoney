# Auth Service

Сервис аутентификации и управления пользователями.

## Описание

Регистрация, аутентификация, управление JWT токенами и профилями.

## Запуск

```bash
make run          # Локально
make test         # Тесты
```

## API примеры

```bash
# Регистрация
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Вход
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```
