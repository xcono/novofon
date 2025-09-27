# Makefile for Novofon API Converter

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=converter
BINARY_UNIX=$(BINARY_NAME)_unix

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./main.go

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Install dependencies
.PHONY: deps
deps:
	$(GOGET) -d -v ./...

# Build for Linux
.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./main.go

# Cross compilation
.PHONY: build-cross
build-cross:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./main.go

# Run the application (requires arguments)
.PHONY: run
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./main.go
	./$(BINARY_NAME)

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  deps         - Install dependencies"
	@echo "  build-linux  - Build for Linux"
	@echo "  run          - Build and run the application"
	@echo "  help         - Show this help"
