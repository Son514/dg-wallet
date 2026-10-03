SHELL := /bin/bash

MIGRATE_PATH ?= database/migrations
MIGRATION_URL := postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable

.PHONY: up down down-all version

up:
	set -a && source .env && set +a && \
	migrate -path $(MIGRATE_PATH) -database "$(MIGRATION_URL)" up

down:
	set -a && source .env && set +a && \
	migrate -path $(MIGRATE_PATH) -database "$(MIGRATION_URL)" down 1

down-all:
	set -a && source .env && set +a && \
	migrate -path $(MIGRATE_PATH) -database "$(MIGRATION_URL)" down -all

version:
	set -a && source .env && set +a && \
	migrate -path $(MIGRATE_PATH) -database "$(MIGRATION_URL)" version