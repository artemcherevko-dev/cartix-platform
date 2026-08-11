FROM golang:1.26.5-alpine AS builder
LABEL authors="artemcherevko"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server /app/server

EXPOSE 3000

ENTRYPOINT ["/app/server"]