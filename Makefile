OS := $(shell uname -s 2>/dev/null || echo Windows)

run: build
	@./bin/api

build:
	@go build -o bin/api cmd/api/main.go

watch:
ifeq (${OS},Windows)
	@air -c .air.win.toml
else
	@air -c .air.toml
endif

migrate:
	@go run cmd/migrate/main.go

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  run       Run the application"
	@echo "  build     Build the application"
	@echo "  watch     Run the application with hot reload"
	@echo "  migrate   Run the migration"
	@echo "  help      Display this help message"

.DEFAULT_GOAL := help