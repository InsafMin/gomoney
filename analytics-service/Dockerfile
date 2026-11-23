# Сборочный образ
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Установка зависимостей для сборки
RUN apk add --no-cache git make

# Копирование go.mod и go.sum
COPY analytics-service/go.mod analytics-service/go.sum ./analytics-service/
COPY shared ./shared

WORKDIR /app/analytics-service

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY analytics-service .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o analytics ./cmd/analytics

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Копирование бинарника из builder
COPY --from=builder /app/analytics-service/analytics .

# Копирование миграций
COPY --from=builder /app/analytics-service/migrations ./migrations

# Установка переменных окружения
ENV DATABASE_URL=""
ENV REDIS_URL=""
ENV KAFKA_BROKERS=""
ENV GRPC_PORT=50052

# Открытие порта
EXPOSE 50052

# Запуск приложения
CMD ["./analytics"]

