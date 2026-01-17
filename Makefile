.PHONY: build test fmt lint quality clean run

# Binary name
BINARY=chaos-kvs

# Build the application
build:
	go build -o $(BINARY) .

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

# Run the application
run: build
	./$(BINARY)
