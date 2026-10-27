.PHONY: all build run test lint clean

BINARY_NAME=docktop
CMD_DIR=./cmd/docktop

all: test build

build:
	go build -o $(BINARY_NAME) $(CMD_DIR)

run:
	go run $(CMD_DIR)

test:
	go test -v -race ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.txt
