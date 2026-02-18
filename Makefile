.RECIPEPREFIX := >
.PHONY: help bootstrap up down logs tidy test run

help:
>@echo "Targets:"
>@echo "  bootstrap  - run initial local setup script"
>@echo "  up         - start mysql/redis with docker compose"
>@echo "  down       - stop docker compose services"
>@echo "  logs       - follow docker compose logs"
>@echo "  tidy       - run go mod tidy"
>@echo "  test       - run go test ./..."
>@echo "  run        - run api entrypoint"

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
