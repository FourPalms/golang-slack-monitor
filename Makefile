.PHONY: help build test run install clean

# Default target
help:
	@echo "Slack Monitor - Available targets:"
	@echo "  make build    - Build the binary"
	@echo "  make test     - Run tests"
	@echo "  make run      - Run the monitor locally"
	@echo "  make install  - Install to ~/bin"
	@echo "  make clean    - Clean build artifacts"

build:
	@echo "Building slack-monitor..."
	go build -o slack-monitor main.go
	@echo "Build complete: ./slack-monitor"

test:
	@echo "Running tests..."
	go test -v -cover ./...

run: build
	@echo "Starting slack-monitor..."
	./slack-monitor

install: build
	@echo "Installing to ~/bin..."
	mkdir -p ~/bin
	cp slack-monitor ~/bin/
	@echo "Installed! Add ~/bin to PATH if not already there."

clean:
	@echo "Cleaning..."
	go clean
	@echo "Clean complete."
