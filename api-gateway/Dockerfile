FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
COPY shared ./shared

WORKDIR /app/api-gateway

RUN go mod download

COPY api-gateway .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway ./cmd/gateway

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/api-gateway/gateway .

EXPOSE 8080

CMD ["./gateway"]
