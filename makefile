.PHONY: help build down restart logs ps clean \
        swarm-init swarm-deploy swarm-stop swarm-logs swarm-ps swarm-scale swarm-update swarm-clean \
        swarm-build test

# Default target
help:
	@echo "================================================"
	@echo "Omni Server - Available Commands"
	@echo "================================================"
	@echo ""
	@echo "Docker Compose (Simple Development):"
	@echo "  make build          - Build and start services with docker-compose"
	@echo "  make down           - Stop docker-compose services"
	@echo "  make restart        - Restart docker-compose services"
	@echo "  make logs           - View docker-compose logs"
	@echo "  make ps             - List docker-compose services"
	@echo ""
	@echo "Docker Swarm (Production-like Testing):"
	@echo "  make swarm-init     - Initialize Docker Swarm"
	@echo "  make swarm-build    - Build images for swarm"
	@echo "  make swarm-deploy   - Deploy stack to local swarm"
	@echo "  make swarm-stop     - Stop swarm stack"
	@echo "  make swarm-logs     - View swarm service logs"
	@echo "  make swarm-ps       - List swarm services and tasks"
	@echo "  make swarm-scale    - Scale user-service to 5 replicas"
	@echo "  make swarm-update   - Update services with latest images"
	@echo "  make swarm-clean    - Remove swarm stack and leave swarm"
	@echo "  make swarm-prune    - Clean up unused Docker resources"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Test the running services"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean          - Remove all containers, images, and volumes"
	@echo ""
	@echo "================================================"

# ================================================
# Docker Compose Commands (Simple Development)
# ================================================

build:
	@echo "Building with docker-compose..."
	docker compose --env-file .env.example up --build -d

down:
	@echo "Stopping docker-compose services..."
	docker compose --env-file .env.example down

restart:
	@echo "Restarting docker-compose services..."
	docker compose --env-file .env.example down
	docker compose --env-file .env.example up --build -d

logs:
	docker compose --env-file .env.example logs -f

ps:
	docker compose --env-file .env.example ps

# ================================================
# Docker Swarm Commands (Production-like Testing)
# ================================================

swarm-init:
	@echo "Initializing Docker Swarm..."
	@if docker info | grep -q "Swarm: active"; then \
		echo "Swarm is already initialized"; \
	else \
		docker swarm init; \
	fi

swarm-build:
	@echo "Building Docker images for swarm..."
	docker build -t localhost:5000/user-service:latest ./1-users
	@echo "Build complete!"

swarm-deploy: swarm-init swarm-build
	@echo "Deploying to Docker Swarm..."
	docker stack deploy -c docker-swarm/stack-local.yaml omni-server
	@echo ""
	@echo "Waiting for services to start..."
	@sleep 10
	@echo ""
	@docker stack services omni-server
	@echo ""
	@echo "Access your service at: http://localhost/api/users"
	@echo "Health check at: http://localhost/health"

swarm-stop:
	@echo "Stopping Docker Swarm stack..."
	docker stack rm omni-server
	@echo "Stack removed!"

swarm-logs:
	@echo "Select a service to view logs:"
	@echo "1. user-service"
	@echo "2. nginx"
	@echo "3. user-redis"
	@echo ""
	@echo "Showing user-service logs (use 'make swarm-logs-nginx' for nginx)..."
	docker service logs -f omni-server_user-service

swarm-logs-nginx:
	docker service logs -f omni-server_nginx

swarm-logs-redis:
	docker service logs -f omni-server_user-redis

swarm-ps:
	@echo "Services:"
	@docker stack services omni-server
	@echo ""
	@echo "Tasks:"
	@docker stack ps omni-server

swarm-scale:
	@echo "Scaling user-service to 5 replicas..."
	docker service scale omni-server_user-service=5
	@sleep 3
	@docker stack services omni-server

swarm-scale-up:
	@echo "Scaling all services up..."
	docker service scale omni-server_user-service=10
	docker service scale omni-server_nginx=3
	@sleep 3
	@docker stack services omni-server

swarm-scale-down:
	@echo "Scaling all services down..."
	docker service scale omni-server_user-service=2
	docker service scale omni-server_nginx=2
	@sleep 3
	@docker stack services omni-server

swarm-update:
	@echo "Rebuilding images..."
	@make swarm-build
	@echo "Updating services..."
	docker service update --force --image localhost:5000/user-service:latest omni-server_user-service
	@echo "Update initiated! Rolling update in progress..."

swarm-clean:
	@echo "Removing swarm stack..."
	-docker stack rm omni-server
	@sleep 5
	@echo "Leaving swarm..."
	-docker swarm leave --force
	@echo "Cleanup complete!"

swarm-prune:
	@echo "Cleaning up Docker resources..."
	@echo "This will remove stopped containers, unused images, and build cache."
	docker system prune -f
	docker builder prune -f
	@echo "Cleanup complete!"
	@echo ""
	@docker system df

# ================================================
# Testing
# ================================================

test:
	@echo "Testing services..."
	@echo ""
	@echo "Testing health endpoint:"
	@curl -s http://localhost/health || echo "Failed to connect"
	@echo ""
	@echo ""
	@echo "Testing user service:"
	@curl -s http://localhost/api/users || echo "Failed to connect"
	@echo ""
	@echo ""
	@echo "Tests complete!"

# ================================================
# Cleanup
# ================================================

clean:
	@echo "Cleaning up all Docker resources..."
	@echo "This will remove containers, networks, and volumes!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker compose --env-file .env.example down -v; \
		docker stack rm omni-server 2>/dev/null || true; \
		sleep 5; \
		docker system prune -f; \
		echo "Cleanup complete!"; \
	else \
		echo "Cleanup cancelled."; \
	fi