# Define the path where go-migrate will be installed
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
GOBIN := ${ROOT_DIR}/.bin

# Define the name of the migration tool binary
MIGRATE_BINARY := migrate
MIGRATE_DIR := $(shell pwd)/migrations

POSTGRES_DB=gateway
POSTGRES_HOST=localhost
POSTGRES_PASSWORD=gateway
POSTGRES_USER=gateway
POSTGRES_PORT=5432


.PHONY: config
config:
	cp .env.default .env

.PHONY: up
up:  .data/adminer-save
	@echo '+==============================================================================+'
	@echo '+ This command starts a local backend with all the required dependencies.      +'
	@echo '+                                                                              +'
	@echo '+                                 [ Endpoints ]                                +'
	@echo '+                                                                              +'
	@echo '+   http://0.0.0.0:8080     (the backend itself)                               +'
	@echo '+   http://localhost:5432   (the postgres database)                            +'
	@echo '+   http://localhost:54321  (the adminer interface)                            +'
	@echo '+                                                                              +'
	@echo '+==============================================================================+'
	@echo
	docker compose up --remove-orphans --build

.PHONY: up-stack
up-stack:
	@echo '+==============================================================================+'
	@echo '+ This command starts a local stack that you can use to work on the backend.   +'
	@echo '+                                                                              +'
	@echo '+                                [ Endpoints ]                                 +'
	@echo '+                                                                              +'
	@echo '+   http://localhost:5432   (the postgres database)                            +'
	@echo '+   http://localhost:54321  (the adminer interface)                            +'
	@echo '+                                                                              +'
	@echo '+==============================================================================+'
	@echo
	docker compose up --remove-orphans --build postgres adminer

.PHONY: down
down:
	docker compose down --remove-orphans

.PHONY: clean
clean: down
	rm -rf .bin .data

.PHONY: up
m-up: $(GOBIN)/migrate
	$(GOBIN)/$(MIGRATE_BINARY) -source file://$(MIGRATE_DIR) -database postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable up

.PHONY: down
m-down: $(GOBIN)/migrate
	$(GOBIN)/$(MIGRATE_BINARY) -path $(shell PWD)./migrations -database postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable down

.PHONY: create-migration
create-migration: $(GOBIN)/migrate
	@read -p "Enter the name of the migration: " NAME; \
	$(GOBIN)/$(MIGRATE_BINARY) create -ext sql -dir $(MIGRATE_DIR) $$NAME

$(GOBIN)/migrate: $(MIGRATE_DIR)
	GOBIN=$(GOBIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

$(MIGRATE_DIR):
	mkdir -p $@

.PHONY: install
install: $(GOBIN)/migrate
	
