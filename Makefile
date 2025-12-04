PROGRAM=m_backuper

.PHONY: build

all: build

build:
	@echo "Building the project..."
	go build -o $(PROGRAM) main.go

run: build
	@echo "Running the project..."
	./$(PROGRAM)

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning up..."
	rm -f $(PROGRAM)
