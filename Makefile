.PHONY: all build build-web test clean run demo

BINARY_NAME=tracereplay
BUILD_DIR=bin

all: build-web build

build-web:
	@echo "Building React 19 web studio..."
	cd web && npm run build
	mkdir -p pkg/server/ui/dist
	cp -r web/dist/* pkg/server/ui/dist/

build:
	@echo "Building TraceReplay standalone binary..."
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tracereplay

test:
	@echo "Running all backend tests..."
	go test -v -race ./...

demo: build
	@echo "Running 15-step autonomous agent scenario..."
	./$(BUILD_DIR)/$(BINARY_NAME) agent run

clean:
	@echo "Cleaning artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf web/dist
	rm -rf pkg/server/ui/dist
	rm -f *.db *.db-journal *.db-wal *.db-shm
