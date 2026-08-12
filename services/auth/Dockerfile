FROM golang:1.26.5-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./

COPY pkg/nats ./pkg/nats
COPY proto ./proto
COPY shared ./shared
COPY services/auth ./services/auth
COPY services/gateway ./services/gateway

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -ldflags="-s -w" \
    -o /app/server \
    ./services/auth/cmd/

FROM alpine:3.20

RUN apk --no-cache add \
    ca-certificates \
    tzdata

WORKDIR /app

COPY --from=builder /app/server ./server

EXPOSE 3000

ENTRYPOINT ["./server"]