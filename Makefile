BINARY_NAME=finance-agent
BUILD_DIR=bin

.PHONY: build test lint clean run dev

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -race -count=1 ./...

test-verbose:
	go test -race -count=1 -v ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

dev:
	go run main.go
