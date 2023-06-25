# Define the path where go-migrate will be installed
GOBIN := $(shell pwd)/.bin

# Define the name of the migration tool binary
MIGRATE_BINARY := migrate
MIGRATE_DIR := $(shell pwd)/migrations

POSTGRES_DB=gateway
POSTGRES_HOST=localhost
POSTGRES_PASSWORD=gateway
POSTGRES_USER=gateway
POSTGRES_PORT=5432

.data:
	mkdir -p $@

.PHONY: start
start: .data
	docker-compose up --remove-orphans 

.PHONY: up
up: $(GOBIN)/migrate
	$(GOBIN)/$(MIGRATE_BINARY) -source file://$(MIGRATE_DIR) -database postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable up

.PHONY: down
down: $(GOBIN)/migrate
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
	
