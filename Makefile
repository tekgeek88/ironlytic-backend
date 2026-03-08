# Load .env.development only if it exists
ifneq ("$(wildcard .env.development)","")
  include .env.development
  export
endif

# ---------- Environment / DB ----------
ENV ?= staging
DB_USER    ?= ironlytic_development_user
DB_PASSWORD ?= ironlytic_password
DB_HOST    ?= localhost
DB_PORT    ?= 5432
DB_NAME    ?= ironlytic_development
DB_SSLMODE ?= disable

DATABASE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

MIGRATE_DIR := ./migrations

# ---------- Build ----------
APP_NAME := ironlytic-api
GOOS         ?= $(shell go env GOOS)
GOARCH       ?= $(shell go env GOARCH)
CGO_ENABLED  ?= 0

VERSION  := $(shell git describe --tags --always --dirty)
COMMIT   := $(shell git rev-parse HEAD)
BUILDTIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := github.com/tekgeek88/ironlytic-backend/version

LD_FLAGS := -X '$(VERSION_PKG).Version=$(VERSION)' \
            -X '$(VERSION_PKG).GitCommit=$(COMMIT)' \
            -X '$(VERSION_PKG).BuildTime=$(BUILDTIME)' \
            -X '$(VERSION_PKG).GoEnv=$(ENV)'

.PHONY: run dev debug build tidy fmt clean \
        migrate-up migrate-down migrate-new print-db-url seed truncate reset-db

run:
	@echo "Running $(APP_NAME) VERSION=$(VERSION) COMMIT=$(COMMIT) ENV=$(ENV)"
	@go run -ldflags "$(LD_FLAGS)" ./cmd/api

dev:
	@echo "Building $(APP_NAME) (local) VERSION=$(VERSION) COMMIT=$(COMMIT) ENV=$(ENV)"
	@go build -o $(APP_NAME) -ldflags "$(LD_FLAGS)" ./cmd/api

debug:
	@echo "Building $(APP_NAME) (debug) VERSION=$(VERSION) COMMIT=$(COMMIT) ENV=$(ENV)"
	@CGO_ENABLED=1 go build -gcflags="all=-N -l" -ldflags "$(LD_FLAGS)" -o $(APP_NAME)-debug ./cmd/api

build:
	@echo "Building $(APP_NAME) VERSION=$(VERSION) COMMIT=$(COMMIT) ENV=$(ENV)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(APP_NAME) -ldflags "$(LD_FLAGS)" ./cmd/api

tidy:
	go mod tidy

fmt:
	go fmt ./...

clean:
	@rm -f $(APP_NAME) $(APP_NAME)-debug
	@rm -rf bin

# ---------- Migrations (requires migrate CLI) ----------
migrate-up:
	migrate -path $(MIGRATE_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATE_DIR) -database "$(DATABASE_URL)" down 1

migrate-new:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATE_DIR) -seq $$name

print-db-url:
	@echo "DATABASE_URL = $(DATABASE_URL)"

# ---------- Seed helpers ----------
seed:
	docker run --rm \
		--network host \
		-v $(PWD)/seed:/seed \
		postgres:16 \
		sh -c 'psql "$(DATABASE_URL)" -f /seed/seed.sql'

truncate:
	docker run --rm \
		--network host \
		-v $(PWD)/seed:/seed \
		postgres:16 \
		sh -c 'psql "$(DATABASE_URL)" -f /seed/truncate.sql'

reset-db: truncate seed
