.PHONY: run up down logs clean

run: bot
	./bot

bot:
	go build -o bot ./cmd/bot

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f bot

clean:
	rm -f bot
