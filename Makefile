up:
	docker compose up --build -d

dev:
	docker compose up --build audiusd-dev -d

stage:
	docker compose up --build audiusd-stage -d

sentry:
	docker compose up -d audiusd-prod-sentry

prod:
	docker compose build --no-cache audiusd-prod && docker compose up -d audiusd-prod

prod-2:
	docker compose up --build audiusd-prod-2 -d

go:
	go run main.go

clean:
	docker compose down
	rm -rf ./tmp
