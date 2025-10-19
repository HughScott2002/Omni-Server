#!/bin/bash
set -e

# DigitalOcean Docker Swarm Setup Script
# This script initializes a Docker Swarm cluster on DigitalOcean

echo "================================================"
echo "Docker Swarm Setup for DigitalOcean"
echo "================================================"

# Configuration
SWARM_MANAGER_IP="${1:-}"
SWARM_WORKERS="${2:-}"

if [ -z "$SWARM_MANAGER_IP" ]; then
    echo "Usage: $0 <manager-ip> [worker-ip-1 worker-ip-2 ...]"
    echo ""
    echo "Example: $0 192.168.1.10 192.168.1.11 192.168.1.12"
    echo ""
    echo "Prerequisites:"
    echo "1. Create 2-3 DigitalOcean droplets with Docker installed"
    echo "2. Set up SSH keys for passwordless access"
    echo "3. Open required firewall ports (see README)"
    exit 1
fi

echo "Manager IP: $SWARM_MANAGER_IP"
echo "Workers: $SWARM_WORKERS"
echo ""

# Initialize Swarm on manager node
echo "Initializing Docker Swarm on manager node..."
ssh root@$SWARM_MANAGER_IP "docker swarm init --advertise-addr $SWARM_MANAGER_IP"

# Get join token for workers
echo "Getting worker join token..."
JOIN_TOKEN=$(ssh root@$SWARM_MANAGER_IP "docker swarm join-token worker -q")
echo "Worker token: $JOIN_TOKEN"

# Join worker nodes to the swarm
if [ -n "$SWARM_WORKERS" ]; then
    for WORKER_IP in $SWARM_WORKERS; do
        echo "Adding worker node: $WORKER_IP"
        ssh root@$WORKER_IP "docker swarm join --token $JOIN_TOKEN $SWARM_MANAGER_IP:2377"
    done
fi

# Verify cluster
echo ""
echo "Verifying cluster status..."
ssh root@$SWARM_MANAGER_IP "docker node ls"

# Create secrets
echo ""
echo "Creating secrets..."
ssh root@$SWARM_MANAGER_IP "echo 'CHANGE_THIS_IN_PRODUCTION' | docker secret create redis_password -"

echo ""
echo "================================================"
echo "Docker Swarm cluster setup complete!"
echo "================================================"
echo ""
echo "Next steps:"
echo "1. Update docker-swarm/.env.production with your DigitalOcean registry"
echo "2. Set up GitHub secrets for CI/CD"
echo "3. Push to main branch to trigger deployment"
echo ""
echo "Manager node: root@$SWARM_MANAGER_IP"
