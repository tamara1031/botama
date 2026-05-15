.PHONY: run up down logs clean test vet fmt

run: bot
	./bot

bot:
	go build -o bot ./cmd/bot

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f bot

clean:
	rm -f bot
