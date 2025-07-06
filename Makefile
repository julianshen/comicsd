.PHONY: build clean test run-mcp install deps fmt vet

# Build the application
build:
	go build -o comicsd ./cmd/comicsd

# Build for production with optimizations
build-prod:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o comicsd ./cmd/comicsd

# Clean build artifacts
clean:
	rm -f comicsd
	rm -f *.cbz *.epub
	go clean

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run MCP server
run-mcp: build
	./comicsd mcp

# Install to system (modify path as needed)
install: build
	cp comicsd /usr/local/bin/

# Create release builds for multiple platforms
release:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o dist/comicsd-darwin-amd64 ./cmd/comicsd
	GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o dist/comicsd-darwin-arm64 ./cmd/comicsd
	GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o dist/comicsd-linux-amd64 ./cmd/comicsd
	GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o dist/comicsd-windows-amd64.exe ./cmd/comicsd

# Run all checks
check: fmt vet test

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  build-prod   - Build with production optimizations"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Install/update dependencies"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  lint         - Run linter"
	@echo "  run-mcp      - Build and run MCP server"
	@echo "  install      - Install to system"
	@echo "  release      - Create release builds"
	@echo "  check        - Run all checks (fmt, vet, test)"