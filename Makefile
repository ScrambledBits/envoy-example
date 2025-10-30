.PHONY: help build up down logs restart test clean health stats

# Docker Compose command (use v2 syntax)
DOCKER_COMPOSE := docker compose

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build all Docker images
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

up: ## Start all services
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d
	@echo "Services started. Access the application at http://localhost:10000"
	@echo "Admin interface available at http://localhost:9901"

down: ## Stop all services
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down

logs: ## Show logs from all services
	$(DOCKER_COMPOSE) logs -f

restart: down up ## Restart all services

test: ## Run basic tests
	@echo "Testing web service through Envoy proxy..."
	@for i in 1 2 3 4 5; do \
		echo "Request $$i:"; \
		curl -s http://localhost:10000; \
		echo ""; \
	done
	@echo ""
	@echo "Testing health endpoint..."
	@curl -s http://localhost:10000/health
	@echo ""
	@echo "Testing liveness endpoint..."
	@curl -s http://localhost:10000/live
	@echo ""
	@echo "Testing readiness endpoint..."
	@curl -s http://localhost:10000/ready | python3 -m json.tool 2>/dev/null || curl -s http://localhost:10000/ready
	@echo ""

health: ## Check health of all services
	@echo "Checking Docker Compose services..."
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "Checking Envoy cluster health..."
	@curl -s http://localhost:9901/clusters | grep -E "health_flags|hostname" || echo "Envoy not responding"

stats: ## Show Envoy statistics
	@curl -s http://localhost:9901/stats | grep -E "upstream_rq_total|upstream_rq_success|upstream_rq_error|health_check"

clean: down ## Stop services and remove volumes
	@echo "Cleaning up..."
	$(DOCKER_COMPOSE) down -v
	docker system prune -f

dev: ## Start services and follow logs
	@make up
	@sleep 5
	@make logs

validate: ## Validate configuration files
	@echo "Validating docker-compose.yml..."
	@$(DOCKER_COMPOSE) config > /dev/null && echo "✓ docker-compose.yml is valid"
	@echo "Validating Envoy configuration..."
	@docker run --rm -v $(PWD)/envoy-proxy/envoy.yaml:/etc/envoy/envoy.yaml envoyproxy/envoy:v1.31-latest --mode validate -c /etc/envoy/envoy.yaml

image-sizes: ## Show Docker image sizes
	@echo "Docker image sizes:"
	@docker images | grep -E "web-service|envoy-proxy|REPOSITORY"
