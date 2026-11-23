FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY transaction-service/go.mod transaction-service/go.sum ./transaction-service/
COPY shared ./shared

WORKDIR /app/transaction-service

RUN go mod download

COPY transaction-service .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o transaction-service ./cmd/transaction

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/transaction-service/transaction-service .
COPY --from=builder /app/transaction-service/migrations ./migrations

EXPOSE 50052

CMD ["./transaction-service"]
