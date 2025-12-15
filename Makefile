.PHONY: build build-android build-windows build-linux test run install-termux clean

# Default build
build:
	@echo "==== Building m_backuper..."
	go build -o m_backuper ./cmd/m_backuper
	@echo "==== Build complete: m_backuper"

# Platform-specific builds
build-android:
	@echo "==== Building for Android (arm64)..."
	GOOS=android GOARCH=arm64 go build -o m_backuper-android ./cmd/m_backuper
	@echo "==== Build complete: m_backuper-android"

build-windows:
	@echo "==== Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -o m_backuper.exe ./cmd/m_backuper
	@echo "==== Build complete: m_backuper.exe"

build-linux:
	@echo "==== Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o m_backuper-linux ./cmd/m_backuper
	@echo "==== Build complete: m_backuper-linux"

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
	cp m_backuper-android $(PREFIX)/bin/m_backuper
	chmod +x $(PREFIX)/bin/m_backuper
	@echo "==== Installation complete"

# Clean build artifacts
clean:
	@echo "==== Cleaning build artifacts..."
	rm -f m_backuper m_backuper-android m_backuper.exe m_backuper-linux
	@echo "==== Clean complete"
