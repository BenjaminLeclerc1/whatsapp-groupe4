.PHONY: all build run test clean docker-build docker-up docker-down

# Variables
GO=go
DOCKER_COMPOSE=docker-compose

# Commandes principales
all: build

build:
	$(GO) build -o bin/api-gateway ./cmd/api-gateway
	$(GO) build -o bin/user-service ./cmd/user-service
	$(GO) build -o bin/message-service ./cmd/message-service
	$(GO) build -o bin/notification-service ./cmd/notification-service
	$(GO) build -o bin/auth-service ./cmd/auth-service
	$(GO) build -o bin/channel-service ./cmd/channel-service

run-gateway:
	$(GO) run ./cmd/api-gateway

run-user:
	$(GO) run ./cmd/user-service

run-message:
	$(GO) run ./cmd/message-service

run-notification:
	$(GO) run ./cmd/notification-service

run-auth:
	$(GO) run ./cmd/auth-service

run-channel:
	$(GO) run ./cmd/channel-service

test:
	$(GO) test -v ./...

clean:
	rm -rf bin/
	$(GO) clean

# Commandes Docker
docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Télécharger les dépendances
deps:
	$(GO) mod download
	$(GO) mod tidy

# Formater le code
fmt:
	$(GO) fmt ./...

# Linter
lint:
	golangci-lint run ./...
