.PHONY: help build test run install clean test-message

# Default target
help:
	@echo "Slack Monitor - Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run unit tests"
	@echo "  make test-message  - Test ntfy.sh notification (requires NTFY_TOPIC)"
	@echo "  make run           - Run the monitor locally (keeps Mac awake)"
	@echo "  make install       - Install to ~/bin"
	@echo "  make clean         - Clean build artifacts"

build:
	@echo "Building slack-monitor..."
	go build -o slack-monitor ./cmd/slack-monitor
	@echo "Build complete: ./slack-monitor"

test:
	@echo "Running tests..."
	go test -v -cover ./...

run: build
	@echo "Starting slack-monitor (keeping Mac awake)..."
	@echo "Press Ctrl+C to stop"
	caffeinate -i ./slack-monitor

install: build
	@echo "Installing to ~/bin..."
	mkdir -p ~/bin
	cp slack-monitor ~/bin/
	@echo "Installed! Add ~/bin to PATH if not already there."

clean:
	@echo "Cleaning..."
	go clean
	@echo "Clean complete."

test-message:
	@echo "Testing ntfy.sh notification..."
	@if [ -z "$(NTFY_TOPIC)" ]; then \
		echo "Error: NTFY_TOPIC environment variable not set"; \
		echo "Usage: NTFY_TOPIC=your-topic make test-message"; \
		exit 1; \
	fi
	@echo "Sending test notification to ntfy.sh/$(NTFY_TOPIC)..."
	@curl -d "Test notification from slack-monitor at $$(date '+%H:%M:%S')" https://ntfy.sh/$(NTFY_TOPIC)
	@echo ""
	@echo "Check your phone for the notification!"
