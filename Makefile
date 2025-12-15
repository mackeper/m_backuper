.PHONY: build build-all build-android build-windows build-linux build-darwin test run install-termux clean

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
test:
	@echo "==== Running tests..."
	go test -v ./...
	@echo "==== Tests complete"

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
