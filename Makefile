.PHONY: build test test-unit test-integration run docker-build docker-up docker-down migrate clean

SRC_DIR := src
DOCKER_DIR := docker
BINARY := $(SRC_DIR)/api

build:
	cd $(SRC_DIR) && go build -o api ./cmd/api

test: test-unit

test-unit:
	cd $(SRC_DIR) && go test ./internal/domain/... ./internal/application/... \
		./internal/infrastructure/github/... ./internal/infrastructure/gitlab/... \
		./internal/presentation/... -count=1

test-integration:
	cd $(SRC_DIR) && \
		DOCKER_HOST=unix:///run/user/$$(id -u)/podman/podman.sock \
		TESTCONTAINERS_RYUK_DISABLED=true \
		go test ./internal/infrastructure/persistence/... -count=1 -v

test-all: test-unit test-integration

run: build
	cd $(SRC_DIR) && ./api

docker-build:
	cd $(DOCKER_DIR) && podman-compose build

docker-up:
	cd $(DOCKER_DIR) && podman-compose up -d

docker-down:
	cd $(DOCKER_DIR) && podman-compose down

docker-logs:
	cd $(DOCKER_DIR) && podman-compose logs -f

migrate:
	cd $(SRC_DIR) && go run ./cmd/api

clean:
	rm -f $(BINARY)
	cd $(SRC_DIR) && go clean ./...
