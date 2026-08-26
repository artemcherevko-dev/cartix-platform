#!/usr/bin/env bash
#
# Run all services locally for development (no Docker).
# Prerequisites: Postgres and NATS reachable at localhost (e.g. only infra from docker-compose).
# Ctrl+C stops everything gracefully (SIGTERM is forwarded to each service).

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$ROOT_DIR/.dev-bin"
mkdir -p "$BIN_DIR"

if [[ -f "$ROOT_DIR/.env" ]]; then
	set -a
	source "$ROOT_DIR/.env"
	set +a
else
	echo "[dev] warning: $ROOT_DIR/.env not found, relying on exported environment" >&2
fi

NAMES=()
PIDS=()

build() {
	local name="$1" dir="$2" path="$3"
	echo "[dev] building ${name}..."
	(cd "$ROOT_DIR/services/$dir" && go build -o "$BIN_DIR/$name" "$path")
}

build gateway gateway ./cmd
build auth auth ./cmd
build profile profile ./cmd
build notification notification ./cmd
build payments payments ./cmd
build order order ./cmd
build catalog catalog ./cmd

start() {
	local name="$1"
	echo "[dev] starting ${name}..."
	"$BIN_DIR/$name" &
	NAMES+=("$name")
	PIDS+=("$!")
}

shutdown() {
	trap - INT TERM EXIT
	echo ""
	echo "[dev] shutting down..."
	for pid in "${PIDS[@]:-}"; do
		kill -TERM "$pid" 2>/dev/null || true
	done
	for pid in "${PIDS[@]:-}"; do
		wait "$pid" 2>/dev/null || true
	done
	echo "[dev] all services stopped"
}
trap shutdown INT TERM EXIT

echo "[dev] starting gateway first..."
start gateway
sleep 1
start auth
start profile
start notification
start payments
sleep 1
start catalog
start order

echo "[dev] all services running. Press Ctrl+C to stop."
while true; do
	sleep 1
	for i in "${!PIDS[@]}"; do
		if ! kill -0 "${PIDS[$i]}" 2>/dev/null; then
			echo "[dev] ${NAMES[$i]} exited unexpectedly, stopping everything..."
			exit 1
		fi
	done
done
