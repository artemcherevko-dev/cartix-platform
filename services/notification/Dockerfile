FROM golang:1.26.5-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./

COPY pkg/nats ./pkg/nats
COPY pkg/redis ./pkg/redis
COPY proto ./proto
COPY shared ./shared
COPY services/auth ./services/auth
COPY services/catalog ./services/catalog
COPY services/gateway ./services/gateway
COPY services/notification ./services/notification
COPY services/order ./services/order
COPY services/payments ./services/payments
COPY services/profile ./services/profile

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -ldflags="-s -w" \
    -o /app/server \
    ./services/notification/cmd/

FROM alpine:3.20

RUN apk --no-cache add \
    ca-certificates \
    tzdata

WORKDIR /app

COPY --from=builder /app/server ./server
COPY --from=builder /app/services/notification/templates ./templates

ENV TEMPLATE_DIR=/app/templates

ENTRYPOINT ["./server"]
