# GoMoney - Финансовый трекер

Микросервисное приложение для управления личными финансами.

## 📋 Описание проекта

GoMoney - это полнофункциональная система управления финансами, построенная на микросервисной архитектуре. Позволяет пользователям отслеживать доходы и расходы, управлять категориями, получать аналитику и контролировать свой бюджет.

## 🏗️ Архитектура

```
┌─────────────┐
│   Клиент    │
│ (Web/Mobile)│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    Nginx    │ :80
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ API Gateway │ :8080 (REST)
└──────┬──────┘
       │
       ├────────────┬────────────┬
       ▼            ▼            ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│   Auth   │ │Transaction│ │Analytics │
│ Service  │ │  Service  │ │  Service │
│  :50051  │ │  :50052   │ │  :50053  │
└────┬─────┘ └─────┬─────┘ └─────┬────┘
     │             │              │
     ▼             ▼              ▼
┌──────────┐ ┌──────────┐   ┌─────────┐
│PostgreSQL│ │PostgreSQL│   │  Kafka  │
│   +      │ │          │   │         │
│  Redis   │ │          │   │         │
└──────────┘ └──────────┘   └─────────┘
```

## 🚀 Микросервисы

### 1. API Gateway (:8080)
**Единая точка входа** для всех клиентских запросов.

**Функции:**
- REST API endpoints
- JWT аутентификация
- Rate limiting (100 req/min)
- Circuit breaker
- CORS поддержка

**Технологии:** Go, Gin, gRPC clients

---

### 2. Auth Service (:50051)
**Сервис аутентификации и авторизации.**

**Функции:**
- Регистрация пользователей
- Вход/выход
- JWT токены (access + refresh)
- Управление профилями
- Валидация токенов

**Технологии:** Go, gRPC, PostgreSQL, Redis

---

### 3. Transaction Service (:50052)
**Управление финансовыми транзакциями.**

**Функции:**
- Создание транзакций (доходы/расходы)
- Фильтрация и поиск
- Расчет баланса
- Управление категориями
- Публикация событий в Kafka

**Технологии:** Go, gRPC, PostgreSQL, Kafka

---

### 4. Analytics Service (:50053)
**Аналитика и отчеты** (опционально, если реализован).

**Функции:**
- Потребление событий из Kafka
- Построение отчетов
- Статистика по категориям
- Трендовый анализ

---

### 5. Shared Module
**Общие protobuf контракты** для всех сервисов.

**Содержит:**
- auth.proto
- transaction.proto
- analytics.proto
- Сгенерированный Go код

## 🛠️ Технологический стек

| Компонент | Технология |
|-----------|------------|
| Язык | Go 1.21 |
| API Gateway | Gin |
| Микросервисы | gRPC |
| Базы данных | PostgreSQL 15 |
| Кеш | Redis 7 |
| Message Queue | Apache Kafka |
| Контейнеризация | Docker, docker-compose |
| Reverse Proxy | Nginx |

## 📦 Структура проекта

```
gomoney/
├── api-gateway/           # REST API Gateway (:8080)
├── auth-service/          # Аутентификация (:50051)
├── transaction-service/   # Транзакции (:50052)
├── analytics-service/     # Аналитика (:50053)
├── shared/                # Protobuf контракты
├── docker-compose.yml     # Оркестрация всех сервисов
├── nginx.conf             # Конфигурация Nginx
└── README.md              # Этот файл
```

## 🚦 Быстрый старт

### Предварительные требования

- Docker и docker-compose
- Go 1.21+ (для разработки)
- protoc (для генерации proto)

### Запуск всей системы

```bash
# Клонировать репозиторий
git clone https://github.com/InsafMin/gomoney.git
cd gomoney

# Запустить все сервисы
docker-compose up -d

# Проверить статус
docker-compose ps

# Просмотр логов
docker-compose logs -f

# Остановить
docker-compose down
```

### Локальная разработка

```bash
# Установить зависимости
cd auth-service && go mod download
cd ../transaction-service && go mod download
cd ../api-gateway && go mod download

# Запустить сервис локально
cd auth-service
make run

# Запустить тесты
make test

# Проверить покрытие
make coverage
```

## 🔌 API Endpoints

### Аутентификация

```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
GET  /api/v1/auth/profile
```

### Транзакции

```http
POST /api/v1/transactions
GET  /api/v1/transactions
GET  /api/v1/transactions/balance
```

### Категории

```http
GET  /api/v1/categories
POST /api/v1/categories
```

## 🧪 Тестирование

### Unit тесты

```bash
# Все сервисы
./scripts/test-all.sh

# Конкретный сервис
cd auth-service && make test
cd transaction-service && make test
```

## 🔒 Безопасность

### Аутентификация
- JWT токены (HMAC-SHA256)
- Access token: 15 минут
- Refresh token: 7 дней
- Хеширование паролей: bcrypt (cost 10)

### Rate Limiting
- 100 запросов в минуту (по IP)
- Настраивается через ENV

### Network Security
- Только port 80/443 наружу (через Nginx)
- Все gRPC порты - internal network
- Docker network isolation

## 🎯 Отказоустойчивость

### Паттерны

1. **Circuit Breaker**
   - Timeout: 5 секунд
   - Failure ratio: 60%
   - Автоматическое восстановление

2. **Retry Mechanism**
   - Kafka producer: 3 попытки
   - Exponential backoff

3. **Graceful Shutdown**
   - Завершение активных запросов
   - Закрытие соединений
   - Timeout: 5-10 секунд

4. **Fallback**
   - Redis failure → прямой запрос к БД
   - Kafka failure → логирование + продолжение

## 📊 Мониторинг и логирование

### Логирование
- Структурированные логи
- Уровни: INFO, WARN, ERROR
- Request ID для трассировки
- Timestamp на каждом событии

### Health Checks
```bash
# API Gateway
curl http://localhost:8080/health

# gRPC сервисы
grpcurl localhost:50051 grpc.health.v1.Health/Check
```

## 🔧 Конфигурация

### Environment переменные

**Auth Service:**
```bash
DB_HOST=localhost
DB_PORT=5432
REDIS_HOST=localhost
JWT_SECRET=
GRPC_PORT=50051
```

**Transaction Service:**
```bash
DB_HOST=localhost
KAFKA_BROKERS=localhost:9092
GRPC_PORT=50052
```

**API Gateway:**
```bash
HTTP_PORT=8080
AUTH_SERVICE_URL=auth-service:50051
TRANSACTION_SERVICE_URL=transaction-service:50052
```

Полный список: см. README каждого сервиса.

## 📈 Масштабирование

### Горизонтальное
- Stateless сервисы
- Load balancing через Nginx
- Kafka партиционирование

### Вертикальное
- Connection pooling
- Resource limits в Docker
- Оптимизация запросов БД

## 🗄️ База данных

### Схема

**Auth Service:**
- users (id, email, password_hash, first_name, last_name)
- refresh_tokens (token_hash, user_id, expires_at)

**Transaction Service:**
- transactions (id, user_id, amount, currency, type, category_id)
- categories (id, user_id, name, type, color, icon)

**Индексы:** Оптимизированы для частых запросов

### Миграции
```bash
# Автоматически при старте каждого сервиса
# Или вручную:
psql -h localhost -U user -d dbname -f migrations/001_init.sql
```

