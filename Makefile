.PHONY: build build-all build-android build-windows build-linux build-darwin test test-unit test-integration test-integration-docker fmt fmt-check lint run install-termux clean clean-test

# Default build (current platform)
build:
	@echo "==== Building m_backuper..."
	@mkdir -p bin
	go build -o bin/m_backuper ./cmd/m_backuper
	@echo "==== Build complete: bin/m_backuper"

# Build all platforms
build-all: build-android build-windows build-linux build-darwin

# Platform-specific builds
build-android:
	@echo "==== Building for Android (arm64)..."
	@mkdir -p bin
	GOOS=android GOARCH=arm64 go build -o bin/m_backuper_android_arm64 ./cmd/m_backuper
	@echo "==== Build complete: bin/m_backuper_android_arm64"

build-windows:
	@echo "==== Building for Windows (amd64)..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/m_backuper_windows_amd64.exe ./cmd/m_backuper
	@echo "==== Build complete: bin/m_backuper_windows_amd64.exe"

build-linux:
	@echo "==== Building for Linux (amd64)..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/m_backuper_linux_amd64 ./cmd/m_backuper
	@echo "==== Build complete: bin/m_backuper_linux_amd64"

build-darwin:
	@echo "==== Building for macOS (amd64)..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/m_backuper_darwin_amd64 ./cmd/m_backuper
	@echo "==== Build complete: bin/m_backuper_darwin_amd64"
	@echo "==== Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build -o bin/m_backuper_darwin_arm64 ./cmd/m_backuper
	@echo "==== Build complete: bin/m_backuper_darwin_arm64"

# Testing
test: test-unit
	@echo "==== All tests complete"

test-unit:
	@echo "==== Running unit tests..."
	go test -v ./...
	@echo "==== Unit tests complete"

test-integration:
	@echo "==== Running integration tests..."
	@echo "Note: This requires an SMB share mounted at SMB_MOUNT"
	@if [ -z "$$SMB_MOUNT" ]; then \
		echo "Error: SMB_MOUNT environment variable not set"; \
		echo "Example: SMB_MOUNT=/mnt/smb make test-integration"; \
		exit 1; \
	fi
	go test -v -tags=integration ./tests/integration/... -count=1
	@echo "==== Integration tests complete"

test-integration-docker:
	@echo "==== Running integration tests in Docker with Samba..."
	@which docker > /dev/null || (echo "Docker not found. Please install Docker." && exit 1)
	@if docker compose version > /dev/null 2>&1; then \
		COMPOSE_CMD="docker compose"; \
	elif which docker-compose > /dev/null 2>&1; then \
		COMPOSE_CMD="docker-compose"; \
	else \
		echo "Docker Compose not found. Please install Docker Compose."; \
		exit 1; \
	fi; \
	$$COMPOSE_CMD -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test-runner; \
	EXIT_CODE=$$?; \
	echo "==== Cleaning up Docker containers..."; \
	$$COMPOSE_CMD -f docker-compose.test.yml down -v; \
	exit $$EXIT_CODE
	@echo "==== Docker integration tests complete"

# Code Quality
fmt:
	@echo "==== Formatting code..."
	go fmt ./...
	@echo "==== Format complete"

fmt-check:
	@echo "==== Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && gofmt -l . && exit 1)
	@echo "==== Format check passed"

lint:
	@echo "==== Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "==== Linting complete"

# Run
run:
	@echo "==== Running m_backuper..."
	go run ./cmd/m_backuper

# Install on Termux
install-termux: build-android
	@echo "==== Installing to Termux..."
	cp bin/m_backuper_android_arm64 $(PREFIX)/bin/m_backuper
	chmod +x $(PREFIX)/bin/m_backuper
	@echo "==== Installation complete"

# Clean build artifacts
clean:
	@echo "==== Cleaning build artifacts..."
	rm -rf bin
	@echo "==== Clean complete"

clean-test:
	@echo "==== Cleaning test artifacts..."
	rm -rf test-results
	@docker compose -f docker-compose.test.yml down -v 2>/dev/null || docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
	@echo "==== Test cleanup complete"
