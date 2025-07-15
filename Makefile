.PHONY: all build test clean run

all: build

build:
	@echo "Building arcgis-utils..."
	@go build -o bin/arcgis-utils ./cmd/arcgis-utils

test:
	@echo "Running tests..."
	@go test ./cmd/arcgis-utils/...

clean:
	@echo "Cleaning up..."
	@rm -f bin/arcgis-utils
	@rm -rf bin
	@rm -rf output # Assuming 'output' is where processed files are stored
	@rm -rf cmd/arcgis-utils/*.test # Remove test executables

run: build
	@echo "Running arcgis-utils..."
	@./bin/arcgis-utils $(ARGS)
