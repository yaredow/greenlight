include .env
export

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: show available make targets
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

## build/api: build API binaries for local and linux/amd64
.PHONY: build/api
build/api:
	@echo "Building cmd/api..."
	go build -ldflags="-s" -o ./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api

# ==================================================================================== #
# DOCKER
# ==================================================================================== #

## docker/up: start Docker services in detached mode
.PHONY: docker/up
docker/up:
	docker compose up -d

## docker/down: stop and remove Docker services
.PHONY: docker/down
docker/down:
	docker compose down

## docker/db/shell: open a psql shell in the db container
.PHONY: docker/db/shell
docker/db/shell:
	docker compose exec db psql -U ${DB_USER} -d ${DB_NAME}

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run API with live reload (installs air if missing)
.PHONY: run/api
run/api: docker/up
	@if ! command -v air > /dev/null; then \
		echo "Installing Air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air

## db/psql: connect to the database with psql
.PHONY: db/psql
db/psql:
	psql ${GREENLIGHT_DB_DSN}

## db/migration/new: create a new SQL migration file
.PHONY: db/migration/new
db/migration/new:
	@echo "Creating a migration file for ${name}"
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migration/up: apply all pending up migrations
.PHONY: db/migration/up
db/migration/up: confirm
	@echo "Running up migrations..."
	migrate -path ./migrations -database ${GREENLIGHT_DB_DSN} up


# ==================================================================================== #
# PRODUCTION
# ==================================================================================== #
production_host_ip = "192.168.122.112"

## production/connect: open an SSH session to the production server
.PHONY: production
production/connect:
	ssh greenlight@${production_host_ip}

## production/deploy/api: deploy API binary, migrations, and config to production
.PHONY: production/deploy/api
production/deploy/api:
	rsync -P ./bin/linux_amd64/api greenlight@${production_host_ip}:~
	rsync -rP --delete ./migrations greenlight@${production_host_ip}:~
	rsync -P ./remote/production/docker-compose.yaml greenlight@${production_host_ip}:~/docker-compose.yaml
	rsync -P ./remote/production/api.service greenlight@${production_host_ip}:~
	rsync -P ./remote/production/Caddyfile greenlight@${production_host_ip}:~
	ssh -t greenlight@${production_host_ip} '\
		set -eu && \
		docker compose -f ~/docker-compose.yaml up -d && \
		. /etc/environment && \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			docker compose -f ~/docker-compose.yaml exec -T db pg_isready -U "$$DB_USER" -d "$$DB_NAME" && break; \
			sleep 1; \
		done && \
		migrate -path ~/migrations -database $$GREENLIGHT_DB_DSN up && \
		sudo mv ~/api.service /etc/systemd/system/ && \
		sudo systemctl daemon-reload && \
		sudo systemctl enable api && \
		sudo systemctl restart api && \
		sudo mv ~/Caddyfile /etc/caddy/ && \
		sudo systemctl restart caddy \
	'

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy/verify Go modules
.PHONY: tidy
tidy:
	@echo 'Formatting .go files...'
	go fmt ./...
	@echo 'Tidying module dependencies...'
	go mod tidy
	go mod verify
	go mod vendor

## audit: run dependency, lint, and test checks
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
