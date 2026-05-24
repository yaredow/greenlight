include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo "Are you sure? [y/N]" && read ans && [ $${ans: -N} = y ]



# ==================================================================================== #
# BUILD
# ==================================================================================== #

.PHONY: build/api
build/api:
	@echo "Building cmd/api..."
	go build -ldflags="-s" -o ./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run cmd/api/*

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql ${GREENLIGHT_DB_DSN}

## db/migration/new: create a new migration file
.PHONY: db/migration/new
db/migration/new:
	@echo "Creating a migration file for ${name}"
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migration/up: run all up migrations
.PHONY: db/migration/up
db/migration/up: confirm
	@echo "Running up migrations..."
	migrate -path ./migrations -database ${GREENLIGHT_DB_DSN} up

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format all .go files and tidy module dependencies
.PHONY: tidy
tidy:
	@echo 'Formatting .go files...'
	go fmt ./...
	@echo 'Tidying module dependencies...'
	go mod tidy
	go mod verify
	go mod vendor

## audit: run quality control checks
.PHONY: audit
audit:
	@echo 'Checking module dependencies...'
	go mod tidy -diff
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...
