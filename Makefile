.PHONY: fmt fmt-check vet lint test build clean

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

clean:
	rm -rf bin/
