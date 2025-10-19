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

# Check if local registry is running
echo "Checking if local registry is running..."
if ! docker ps | grep -q registry:2; then
    echo "Starting local Docker registry on port 5000..."
    docker run -d -p 5000:5000 --name registry registry:2 || true
    sleep 2
fi

echo ""

# Build user-service
echo "Building user-service..."
docker build -t $REGISTRY/user-service:$VERSION ./1-users
docker tag $REGISTRY/user-service:$VERSION $REGISTRY/user-service:latest
docker push $REGISTRY/user-service:$VERSION
docker push $REGISTRY/user-service:latest

# Build wallet-service
echo "Building wallet-service..."
docker build -t $REGISTRY/wallet-service:$VERSION ./3-wallet
docker tag $REGISTRY/wallet-service:$VERSION $REGISTRY/wallet-service:latest
docker push $REGISTRY/wallet-service:$VERSION
docker push $REGISTRY/wallet-service:latest

# Build notification-service
echo "Building notification-service..."
docker build -t $REGISTRY/notification-service:$VERSION ./2-notification --target production
docker tag $REGISTRY/notification-service:$VERSION $REGISTRY/notification-service:latest
docker push $REGISTRY/notification-service:$VERSION
docker push $REGISTRY/notification-service:latest

# Build transaction-service
echo "Building transaction-service..."
docker build -t $REGISTRY/transaction-service:$VERSION ./4-transactions
docker tag $REGISTRY/transaction-service:$VERSION $REGISTRY/transaction-service:latest
docker push $REGISTRY/transaction-service:$VERSION
docker push $REGISTRY/transaction-service:latest

# Build fraud-detection-service
echo "Building fraud-detection-service..."
docker build -t $REGISTRY/fraud-detection-service:$VERSION ./5-fraud-detection
docker tag $REGISTRY/fraud-detection-service:$VERSION $REGISTRY/fraud-detection-service:latest
docker push $REGISTRY/fraud-detection-service:$VERSION
docker push $REGISTRY/fraud-detection-service:latest

# Nginx uses official image
echo "Nginx will use official image: nginx:1.27.2-alpine"

echo ""
echo "================================================"
echo "Build complete!"
echo "================================================"
echo ""
echo "Images built and pushed to $REGISTRY:"
echo "  $REGISTRY/user-service:$VERSION"
echo "  $REGISTRY/wallet-service:$VERSION"
echo "  $REGISTRY/notification-service:$VERSION"
echo "  $REGISTRY/transaction-service:$VERSION"
echo "  $REGISTRY/fraud-detection-service:$VERSION"
echo ""
echo "To test locally with Docker Swarm:"
echo "  1. Initialize swarm (if not already): docker swarm init"
echo "  2. Deploy stack: ./docker-swarm/deploy-local.sh"
echo ""
