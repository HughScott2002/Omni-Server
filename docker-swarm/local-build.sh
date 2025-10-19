#!/bin/bash
set -e

# Build images locally for testing
# Simulates what CI/CD will do

echo "================================================"
echo "Building Docker images locally"
echo "================================================"

# Load environment variables
if [ -f docker-swarm/.env.production ]; then
    export $(cat docker-swarm/.env.production | grep -v '^#' | xargs)
fi

VERSION="${1:-latest}"
REGISTRY="${DOCKER_REGISTRY:-localhost:5000}"

echo "Building version: $VERSION"
echo "Registry: $REGISTRY"
echo ""

# Build user-service
echo "Building user-service..."
docker build -t $REGISTRY/user-service:$VERSION ./1-users
docker tag $REGISTRY/user-service:$VERSION $REGISTRY/user-service:latest

# Build nginx (using official image, but you can customize if needed)
echo "Nginx will use official image: nginx:1.27.2-alpine"

echo ""
echo "================================================"
echo "Build complete!"
echo "================================================"
echo ""
echo "Images built:"
echo "  $REGISTRY/user-service:$VERSION"
echo "  $REGISTRY/user-service:latest"
echo ""
echo "To test locally with Docker Swarm:"
echo "  1. Initialize swarm: docker swarm init"
echo "  2. Deploy stack: ./docker-swarm/deploy-local.sh"
