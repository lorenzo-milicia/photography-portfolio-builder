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

# Serve the generated static site
serve-static:
	@just build
	@echo "Serving static site on http://localhost:8000"
	./bin/builder serve
