#!/bin/bash
set -e

# Deploy script for Docker Swarm
# This can be run manually or by CI/CD

echo "================================================"
echo "Deploying Omni Server to Docker Swarm"
echo "================================================"

# Load environment variables
if [ -f docker-swarm/.env.production ]; then
    export $(cat docker-swarm/.env.production | grep -v '^#' | xargs)
fi

# Check if we're deploying to remote or local swarm
MANAGER_HOST="${SWARM_MANAGER_HOST:-localhost}"

echo "Deploying to: $MANAGER_HOST"
echo "Registry: $DOCKER_REGISTRY"
echo "Version: $VERSION"
echo ""

# Deploy the stack
if [ "$MANAGER_HOST" = "localhost" ]; then
    # Local deployment
    docker stack deploy -c docker-swarm/stack.yaml omni-server
else
    # Remote deployment (used by CI/CD)
    ssh root@$MANAGER_HOST "docker stack deploy -c /root/omni-server/docker-swarm/stack.yaml omni-server"
fi

echo ""
echo "Deployment initiated!"
echo ""
echo "Check deployment status:"
echo "  docker stack services omni-server"
echo ""
echo "Check logs:"
echo "  docker service logs omni-server_user-service"
echo ""
echo "Scale services:"
echo "  docker service scale omni-server_user-service=5"
