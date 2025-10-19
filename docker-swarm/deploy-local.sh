#!/bin/bash
set -e

# Deploy to local Docker Swarm for testing
# Use this to test before pushing to production

echo "================================================"
echo "Deploying to LOCAL Docker Swarm"
echo "================================================"

# Check if swarm is initialized
if ! docker info | grep -q "Swarm: active"; then
    echo "Initializing Docker Swarm..."
    docker swarm init
fi

# Set local environment
export DOCKER_REGISTRY=localhost:5000
export VERSION=latest
export ENVIRONMENT=local
export MODE=memcached
export USER_REDIS_PASSWORD=sadasdasdasddsfwerweraewrsd34

echo "Deploying stack..."
docker stack deploy -c docker-swarm/stack-local.yaml omni-server

echo ""
echo "================================================"
echo "Local deployment complete!"
echo "================================================"
echo ""
echo "Useful commands:"
echo "  docker stack services omni-server          # List services"
echo "  docker stack ps omni-server               # List tasks"
echo "  docker service logs -f omni-server_user-service   # View logs"
echo "  docker service scale omni-server_user-service=5   # Scale service"
echo "  docker stack rm omni-server               # Remove stack"
echo ""
echo "Access your service at: http://localhost/api/users"
