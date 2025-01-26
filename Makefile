up:
	docker compose up --build -d

dev:
	docker compose up --build audiusd-dev -d

stage:
	docker compose up --build audiusd-stage -d

prod:
	docker compose up --build audiusd-prod -d

clean:
	docker compose down
	rm -rf ./tmp
