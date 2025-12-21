ENV ?= dev

API_NAME ?= snowgo-service
API_IMAGE ?= snowgo
API_VERSION ?= 1.0.0
API_PORT ?= 8000

CONSUMER_NAME ?= snowgo-consumer-service
CONSUMER_IMAGE ?= snowgo-consumer
CONSUMER_VERSION ?= 1.0.0

COMPOSE_FILE ?= docker-compose.yml

.PHONY: help
help: ## Show this help
	@awk 'BEGIN { \
		FS = ":.*##"; \
		printf "\n\033[1mUsage:\033[0m\n  make \033[36m<target>\033[0m\n\n"; \
		printf "\033[1mTargets:\033[0m\n"; \
	} \
	/^[a-zA-Z0-9_-]+:.*##/ { \
		printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 \
	}' $(MAKEFILE_LIST)

.PHONY: mysql-init
mysql-init:  ## Init database schema/data
	@echo "开始初始化数据库..."
	@go run ./internal/dal/cmd/init/main.go

.PHONY: mq-init
mq-init:  ## Init rabbitmq declare
	@echo "开始初始化rabbitmq声明..."
	@go run ./cmd/mq-declarer

.PHONY: api-build
api-build:  ## Build API docker image
	@echo "Building API image..."
	docker build -t $(API_IMAGE):$(API_VERSION) .

.PHONY: api-run
api-run:  api-stop  ## Run API container
	@echo "Running API container..."
	docker run -d \
		--restart unless-stopped \
		--name $(API_NAME) \
		-e ENV=$(ENV) \
		-v ./config:/app/config \
		-v ./logs:/app/logs \
		-p $(API_PORT):$(API_PORT) \
		$(API_IMAGE):$(API_VERSION)

.PHONY: api-stop
api-stop:  ## Stop API container
	@echo "Stopping API container..."
	@docker stop $(API_NAME) 2>/dev/null || true
	@docker rm $(API_NAME) 2>/dev/null || true

.PHONY: consumer-build
consumer-build:  ## Build consumer docker image
	@echo "Building consumer image..."
	docker build -f Dockerfile.consumer \
		-t $(CONSUMER_IMAGE):$(CONSUMER_VERSION) .

.PHONY: consumer-run
consumer-run:  consumer-stop  ## Run consumer container
	@echo "Running consumer container..."
	docker run -d \
		--restart unless-stopped \
		--name $(CONSUMER_NAME) \
		-e ENV=$(ENV) \
		-v ./config:/app/config \
		-v ./logs:/app/logs \
		$(CONSUMER_IMAGE):$(CONSUMER_VERSION)

.PHONY: consumer-stop
consumer-stop:    ## Stop consumer container
	@echo "Stopping consumer container..."
	docker stop $(CONSUMER_NAME) 2>/dev/null || true
	docker rm $(CONSUMER_NAME) 2>/dev/null || true

.PHONY: up
up:  ## Start all services via docker-compose
	docker compose -f $(COMPOSE_FILE) up -d

.PHONY: restart
restart:  ## Restart all services via docker-compose
	docker compose -f $(COMPOSE_FILE) restart

.PHONY: down
down: ## Stop all services via docker-compose
	docker compose -f $(COMPOSE_FILE) down

.PHONY: start stop
start: up
stop: down

.PHONY: test
test:
	go test ./... -cover


# 生成model
.PHONY: gen
gen: do ?= init
gen:  ## Generate DAL code
	go run ./internal/dal/cmd/gen/main.go $(do) && make gen-query
	# git add ./internal/dal/

.PHONY: gen-query
gen-query:  ## Generate Query code
	go run ./internal/dal/cmd/gen/main.go query
