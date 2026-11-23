# Этап сборки
FROM golang:1.24-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git

# Рабочая директория
WORKDIR /app

# Копирование go mod файлов
COPY auth-service/go.mod auth-service/go.sum ./auth-service/
COPY shared ./shared

WORKDIR /app/auth-service

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY auth-service .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-service ./cmd/auth

# Финальный образ
FROM alpine:latest

# Установка ca-certificates для HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копирование бинарного файла из builder
COPY --from=builder /app/auth-service/auth-service .

# Копирование миграций
COPY --from=builder /app/auth-service/migrations ./migrations

# Порт gRPC
EXPOSE 50051

# Запуск приложения
CMD ["./auth-service"]
