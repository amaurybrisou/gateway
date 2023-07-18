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

.PHONY: lint
lint:
	@echo "> Lint backend..."
	golangci-lint run ./...

.PHONY: test
test:
	@echo "> Test backend..."
	ENV=test go test -count=1 -timeout 3m -v ./...

.PHONY: config
config:
	cp .env.default .env

.PHONY: build
build:
	@echo "> Build backend..."
	go build -ldflags=" \
		-X 'github.com/amaurybrisou/gateway/src.BuildHash=$(shell git describe --abbrev=0 --tags)' \
		-X 'github.com/amaurybrisou/gateway/src.BuildHash=$(shell git describe --always --long --dirty)' \
		-X 'github.com/amaurybrisou/gateway/src.BuildTime=$(shell date)'" \
		-o $(GOBIN)/backend cmd/gateway/main.go 

.PHONY: build-docker
build-docker:
	NODE_ENV=production docker build --platform linux/amd64 -t "docker.io/brisouamaury/gateway:latest" --push .

.data/adminer-save:
	mkdir -p $@ && chmod o+w $@

.PHONY: up
up:  #.data/adminer-save
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

.PHONY: m-up
m-up: $(GOBIN)/migrate
	$(GOBIN)/$(MIGRATE_BINARY) -source file://$(MIGRATE_DIR) -database postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable up

.PHONY: m-down
m-down: $(GOBIN)/migrate
	$(GOBIN)/$(MIGRATE_BINARY) -path $(ROOT_DIR)/migrations -database postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable down

.PHONY: m-create
m-create: $(GOBIN)/migrate
	@read -p "Enter the name of the migration: " NAME; \
	$(GOBIN)/$(MIGRATE_BINARY) create -ext sql -dir $(MIGRATE_DIR) $$NAME

$(GOBIN)/migrate: $(MIGRATE_DIR)
	GOBIN=$(GOBIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

$(MIGRATE_DIR):
	mkdir -p $@

.PHONY: install
install: $(GOBIN)/migrate

${GOBIN}/go-callvis:
	GOBIN=${GOBIN} go install github.com/ofabry/go-callvis@latest

.PHONY: tools
tools: ${GOBIN}/go-callvis

.PHONY: callvis
callvis: tools
	${GOBIN}/go-callvis cmd/gateway/main.go

${GOBIN}/govulncheck:
	GOBIN=${GOBIN} go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: check
check: tools
	$(GOBIN)/govulncheck ./...