.PHONY: all build run test lint vet check clean

BINARY_NAME=docktop
CMD_DIR=./cmd/docktop

all: check build

build:
	go build -o $(BINARY_NAME) $(CMD_DIR)

run:
	go run $(CMD_DIR)

test:
	go test -v -race ./...

lint:
	golangci-lint run

vet:
	go vet ./...

check: lint test build

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.txt