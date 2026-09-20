.PHONY: fmt fmt-check vet lint test build clean compose-up compose-down migrate-up migrate-down

# gofmt -l lists files that ARE NOT formatted; `fmt` fixes them, `fmt-check`
# (used in CI) fails if the list is non-empty without touching anything.
fmt:
	gofmt -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./... -race -cover

build:
	go build -o bin/shipyard-api ./cmd/shipyard-api
	go build -o bin/shipyard-worker ./cmd/shipyard-worker

clean:
	rm -rf bin/

# Local Postgres (port 5433, see deployments/compose/docker-compose.yml).
DATABASE_URL ?= postgres://shipyard:shipyard@localhost:5433/shipyard?sslmode=disable

compose-up:
	docker compose -f deployments/compose/docker-compose.yml up -d --wait

compose-down:
	docker compose -f deployments/compose/docker-compose.yml down

# Needs the migrate CLI: go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1
