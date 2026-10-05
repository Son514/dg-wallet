SHELL := /bin/bash

MIGRATE_PATH ?= database/migrations
MIGRATION_URL := postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable

PROTO_DIR  ?= ../auth-service/proto
PROTO_FILE := $(PROTO_DIR)/auth.proto
MODULE     := $(shell go list -m)
GEN_PKG    := $(MODULE)/gen/auth;authv1

.PHONY: up down down-all version proto

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

proto:
	protoc -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=$(MODULE) --go_opt=Mauth.proto="$(GEN_PKG)" \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) --go-grpc_opt=Mauth.proto="$(GEN_PKG)" \
		$(PROTO_FILE)
