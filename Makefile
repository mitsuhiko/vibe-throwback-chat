.PHONY: install dev format check build tail-log

# Install dependencies for both Go and web components
install:
	go mod tidy
	cd web && npm install

# Start development environment with live reloading
dev: install
	./scripts/shoreman.sh

# Format code using Go and npm formatters
format:
	go fmt ./...
	cd web && npm run format

# Run static analysis and type checking
check:
	go vet ./...
	cd web && npm run typecheck

# Build the server binary
build:
	go build -o bin/server ./cmd/server

# Display the last 100 lines of development log with ANSI codes stripped
tail-log:
	@tail -100 ./dev.log | perl -pe 's/\e\[[0-9;]*m(?:\e\[K)?//g'
