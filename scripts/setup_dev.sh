#!/usr/bin/env bash

set -euo pipefail

PROJECT_NAME="${PROJECT_NAME:-stock-bot}"
GO_MODULE="${GO_MODULE:-github.com/example/${PROJECT_NAME}}"
GO_VERSION="${GO_VERSION:-1.23}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "[ERROR] Required command not found: $cmd"
    exit 1
  fi
}

write_if_missing() {
  local path="$1"
  local content="$2"

  if [[ -f "$path" ]]; then
    echo "[SKIP] $path already exists"
    return
  fi

  printf "%b" "$content" >"$path"
  echo "[OK] created $path"
}

echo "[INFO] checking required commands..."
require_cmd go
require_cmd docker

echo "[INFO] preparing directories..."
mkdir -p \
  "$ROOT_DIR/cmd/api" \
  "$ROOT_DIR/internal/app" \
  "$ROOT_DIR/internal/config" \
  "$ROOT_DIR/internal/transport/http" \
  "$ROOT_DIR/migrations" \
  "$ROOT_DIR/scripts"

if [[ ! -f "$ROOT_DIR/go.mod" ]]; then
  echo "[INFO] initializing go module: $GO_MODULE"
  (cd "$ROOT_DIR" && go mod init "$GO_MODULE")
else
  echo "[SKIP] go.mod already exists"
fi

write_if_missing "$ROOT_DIR/.env.example" "APP_ENV=local
APP_PORT=8080
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=stock
DB_PASSWORD=stockpass
DB_NAME=stockbot
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
"

write_if_missing "$ROOT_DIR/docker-compose.yml" "services:
  mysql:
    image: mysql:8.4
    container_name: stockbot-mysql
    restart: unless-stopped
    environment:
      MYSQL_DATABASE: stockbot
      MYSQL_USER: stock
      MYSQL_PASSWORD: stockpass
      MYSQL_ROOT_PASSWORD: rootpass
    ports:
      - \"3306:3306\"
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: [\"CMD\", \"mysqladmin\", \"ping\", \"-h\", \"127.0.0.1\", \"-uroot\", \"-prootpass\"]
      interval: 10s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7.2
    container_name: stockbot-redis
    restart: unless-stopped
    ports:
      - \"6379:6379\"
    volumes:
      - redis-data:/data
    healthcheck:
      test: [\"CMD\", \"redis-cli\", \"ping\"]
      interval: 10s
      timeout: 5s
      retries: 10

volumes:
  mysql-data:
  redis-data:
"

write_if_missing "$ROOT_DIR/Makefile" ".RECIPEPREFIX := >
.PHONY: help bootstrap up down logs tidy test run

help:
>@echo \"Targets:\"
>@echo \"  bootstrap  - run initial local setup script\"
>@echo \"  up         - start mysql/redis with docker compose\"
>@echo \"  down       - stop docker compose services\"
>@echo \"  logs       - follow docker compose logs\"
>@echo \"  tidy       - run go mod tidy\"
>@echo \"  test       - run go test ./...\"
>@echo \"  run        - run api entrypoint\"

bootstrap:
>bash scripts/setup_dev.sh

up:
>docker compose up -d

down:
>docker compose down

logs:
>docker compose logs -f

tidy:
>go mod tidy

test:
>go test ./...

run:
>go run ./cmd/api
"

write_if_missing "$ROOT_DIR/cmd/api/main.go" "package main

import \"fmt\"

func main() {
\tfmt.Println(\"stock-bot bootstrap complete\")
}
"

echo "[INFO] running go mod tidy..."
(cd "$ROOT_DIR" && go mod tidy)

echo
echo "[DONE] local bootstrap complete"
echo "- module: $GO_MODULE"
echo "- next: cp .env.example .env && make up && make run"
