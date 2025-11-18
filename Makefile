.PHONY: build run test clean help install lint

BINARY_NAME=tagTonic
BUILD_DIR=build
GO=go
GOFLAGS=-v

help:
	@echo "Available targets:"
	@echo "  make build      - Build the project"
	@echo "  make run        - Build and run the TUI"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make install    - Install dependencies"
	@echo "  make lint       - Run linters (if available)"

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

run: build
	./$(BUILD_DIR)/$(BINARY_NAME) tui

clean:
	$(GO) clean
	rm -rf ./$(BUILD_DIR)

install:
	$(GO) mod tidy
	$(GO) mod download

lint:
	$(GO) vet ./...
