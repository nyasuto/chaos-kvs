.PHONY: build test fmt lint quality clean run server demo

# Binary name
BINARY=chaos-kvs

# Default server address
ADDR=:8080

# Build the application
build:
	go build -o $(BINARY) ./cmd/chaos-kvs

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run linter (if golangci-lint is installed)
lint:
	golangci-lint run

# Run all quality checks (required before commit)
quality: fmt test lint

# Clean build artifacts
clean:
	rm -f $(BINARY)
	go clean

# Run the application (CLI mode)
run: build
	./$(BINARY)

# Run Web UI server
server: build
	./$(BINARY) --server --addr $(ADDR)

# Run quick demo scenario
demo: build
	./$(BINARY) --preset quick
