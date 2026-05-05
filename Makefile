.PHONY: run up down logs build

run:
	go run ./cmd/bot

build:
	docker compose build --no-cache

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f bot
