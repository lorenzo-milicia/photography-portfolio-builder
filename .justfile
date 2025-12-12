# Justfile generated from Makefile
# Usage: just <recipe> [ARGS]

# default
default: build

# Build the application
build:
	@echo "Building photography portfolio builder..."
	go build -o bin/builder ./cmd/builder
	@echo "Build complete: bin/builder"

# Run the builder server (builds first)
run:
	@just build
	@echo "Starting photography portfolio builder (foreground)..."
	./bin/builder builder

# Run builder on custom port
run-port:
	@just build
	@echo "Starting photography portfolio builder on port ${PORT:-8080}"
	./bin/builder builder -port ${PORT:-8080}

# Generate static site for production (no base URL)
generate-prod:
	@just build
	@echo "Generating static site for production..."
	./bin/builder generate -base-url ""
	@echo "Site generated in output/public/"

# Generate static site for preview (with /preview base URL)
generate-preview:
	@just build
	@echo "Generating static site for preview..."
	./bin/builder generate -base-url "/preview"
	@echo "Site generated in output/public/"

# Serve the generated static site
serve-static:
	@just build
	@echo "Serving static site on http://localhost:8000"
	./bin/builder serve

# Restart server (kills any existing builder process then starts)
restart:
	@echo "Restarting builder..."
	pkill -f './bin/builder' || true
	sleep 1
	./bin/builder builder

# Clean build artifacts
clean:
	rm -rf bin/
	@echo "Clean complete"

# Clean everything including generated content
clean-all: clean
	rm -rf output/public/*
	@echo "Clean all complete"

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Lint (basic)
lint:
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Install dependencies
deps:
	go get -v ./...
	go mod tidy

# Trigger static generation (calls builder API)
# Requires server running locally
generate:
	@echo "Triggering site generation (POST /api/generate)"
	curl -s -X POST http://localhost:8080/api/generate || true

# Development note
dev:
	@echo "For live reload install 'air' (https://github.com/cosmtrek/air) or similar."
