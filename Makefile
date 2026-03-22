.PHONY: build run-mcp run-serve test clean install

BINARY=mio
BUILD_DIR=./bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/mio

run-mcp: build
	$(BUILD_DIR)/$(BINARY) mcp

run-serve: build
	$(BUILD_DIR)/$(BINARY) serve

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f ~/.mio/mio.db

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

# Quick test: save and search
demo: build
	@echo "=== Saving test memories ==="
	$(BUILD_DIR)/$(BINARY) save "Configured auth with JWT" "What: Set up JWT authentication\nWhy: Need secure API access\nWhere: internal/auth/\nLearned: RS256 is preferred over HS256 for multi-service" --type decision --project demo
	$(BUILD_DIR)/$(BINARY) save "Fixed null pointer in user handler" "What: Fixed NPE when user.Email is nil\nWhy: Crash in production on new registrations\nWhere: internal/handlers/user.go:42\nLearned: Always check optional fields before dereferencing" --type bugfix --project demo
	$(BUILD_DIR)/$(BINARY) save "Adopted repository pattern" "What: Introduced repository pattern for data access\nWhy: Decouple business logic from database\nWhere: internal/repository/\nLearned: Interface-based repos make testing much easier" --type architecture --project demo
	@echo ""
	@echo "=== Searching for 'auth' ==="
	$(BUILD_DIR)/$(BINARY) search auth
	@echo "=== Stats ==="
	$(BUILD_DIR)/$(BINARY) stats
