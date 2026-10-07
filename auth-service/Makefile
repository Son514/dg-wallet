SHELL := /bin/bash

MIGRATE_PATH ?= database/migrations
MIGRATION_URL := postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable

PROTO_DIR   ?= proto
MODULE      := $(shell go list -m)
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
WALLET_PROTO ?= ../wallet-service/proto/wallet.proto
WALLET_GEN_PKG := $(MODULE)/gen/wallet;walletv1

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
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		$(PROTO_FILES)
	protoc -I ../wallet-service/proto \
		--go_out=. --go_opt=module=$(MODULE) --go_opt=Mwallet.proto="$(WALLET_GEN_PKG)" \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) --go-grpc_opt=Mwallet.proto="$(WALLET_GEN_PKG)" \
		$(WALLET_PROTO)