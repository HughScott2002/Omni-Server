# Omni Server Deployment Guide

Complete deployment instructions for both local development and VPS production environments.

## Table of Contents

- [Overview](#overview)
- [Local Development (Docker Compose)](#local-development-docker-compose)
- [Local Testing (Docker Swarm)](#local-testing-docker-swarm)
- [VPS Production (Docker Swarm)](#vps-production-docker-swarm)
- [Troubleshooting](#troubleshooting)

## Overview

**Services:**
- Nginx (Load Balancer/Reverse Proxy)
- Kafka (Message Broker)
- User Service (Go)
- Wallet Service (Go)
- Notification Service (Python/FastAPI)
- Transaction Service (Go)
- Fraud Detection Service (Go)
- Redis instances (for caching/persistence)

**Ports:**
- 80: Nginx (public access)
- 8081: User Service (direct access)
- 8082: Wallet Service (direct access)
- 8083: Notification Service (direct access)
- 8084: Transaction Service (direct access)
- 8085: Fraud Detection Service (direct access)

---

## Local Development (Docker Compose)

Use docker-compose for rapid development and testing.

### Prerequisites

- Docker Desktop or Docker Engine (20.10+)
- Docker Compose (2.0+)
- At least 4GB RAM available for Docker

### Step 1: Clone and Setup

```bash
cd /path/to/Omni-Server
```

### Step 2: Configure Environment

```bash
# Copy environment file
cp .env.example .env

# Edit .env if needed (optional for local development)
nano .env
```

### Step 3: Start All Services

```bash
# Build and start all services
docker-compose up --build -d

# Or start specific services
docker-compose up -d user-service wallet-service
```

### Step 4: Verify Services

```bash
# Check all services are running
docker-compose ps

# Check logs
docker-compose logs -f

# Test user service
curl http://localhost/api/users/health

# Test wallet service
curl http://localhost/api/wallets/health

# Test transaction service
curl http://localhost/api/transactions/health

# Test fraud detection
curl http://localhost/api/fraud-detection/health
```

### Step 5: Stop Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes all data)
docker-compose down -v
```

### Common Commands

```bash
# View logs for specific service
docker-compose logs -f user-service

# Restart a service
docker-compose restart wallet-service

# Rebuild a specific service
docker-compose up -d --build transaction-service

# Execute command in container
docker-compose exec user-service sh
```

---

## Local Testing (Docker Swarm)

Test production-like deployment locally before pushing to VPS.

### Prerequisites

- Docker Engine (20.10+)
- At least 6GB RAM available for Docker

### Step 1: Initialize Docker Swarm

```bash
# Initialize swarm (if not already initialized)
docker swarm init

# Verify swarm is active
docker info | grep Swarm
```

### Step 2: Build Images

```bash
# Make build script executable
chmod +x docker-swarm/local-build.sh

# Build all service images
./docker-swarm/local-build.sh

# This will:
# - Start a local Docker registry on port 5000
# - Build all service images
# - Push images to local registry
```

### Step 3: Deploy Stack

```bash
# Make deploy script executable
chmod +x docker-swarm/deploy-local.sh

# Deploy the stack
./docker-swarm/deploy-local.sh
```

### Step 4: Monitor Deployment

```bash
# List all services
docker stack services omni-server

# Check service status
docker stack ps omni-server

# View logs for a service
docker service logs -f omni-server_user-service
docker service logs -f omni-server_transaction-service
docker service logs -f omni-server_fraud-detection-service

# Check specific service details
docker service inspect omni-server_user-service
```

### Step 5: Test the Deployment

```bash
# Test nginx health
curl http://localhost/health

# Test services through nginx
curl http://localhost/api/users/health
curl http://localhost/api/wallets/health
curl http://localhost/api/transactions/health
curl http://localhost/api/fraud-detection/health
```

### Step 6: Scale Services

```bash
# Scale a service up
docker service scale omni-server_user-service=5

# Scale multiple services
docker service scale \
  omni-server_user-service=5 \
  omni-server_wallet-service=5 \
  omni-server_transaction-service=3
```

### Step 7: Remove Stack

```bash
# Remove the entire stack
docker stack rm omni-server

# Wait for all services to be removed
docker stack ps omni-server

# Leave swarm mode (if needed)
docker swarm leave --force
```

---

## VPS Production (Docker Swarm)

Deploy to a production VPS (DigitalOcean, AWS, etc.)

### Prerequisites

- VPS with Ubuntu 20.04+ or Debian 11+
- At least 4GB RAM, 2 CPUs
- Docker Engine installed on VPS
- SSH access to VPS
- Domain name (optional but recommended)

### Step 1: Prepare VPS

SSH into your VPS:

```bash
ssh root@your-vps-ip
```

Install Docker (if not installed):

```bash
# Update system
apt-get update && apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Verify installation
docker --version
```

Initialize Docker Swarm:

```bash
docker swarm init

# If multiple IPs, specify which one to use
docker swarm init --advertise-addr <VPS_PUBLIC_IP>
```

### Step 2: Set Up Docker Registry (Choose One Option)

**Option A: Use Docker Hub (Recommended)**

```bash
# On your local machine, login to Docker Hub
docker login

# Tag and push images
docker tag localhost:5000/user-service:latest yourdockerhub/user-service:latest
docker push yourdockerhub/user-service:latest

# Repeat for all services
```

**Option B: Set Up Private Registry on VPS**

```bash
# On VPS, start a registry
docker run -d -p 5000:5000 --name registry \
  --restart=always \
  -v /mnt/registry:/var/lib/registry \
  registry:2

# Enable insecure registry (for HTTP)
# Edit /etc/docker/daemon.json
cat > /etc/docker/daemon.json <<EOF
{
  "insecure-registries": ["your-vps-ip:5000"]
}
EOF

# Restart Docker
systemctl restart docker
```

### Step 3: Update Environment Configuration

On VPS, create `.env.production`:

```bash
cd /root
mkdir omni-server
cd omni-server

# Create environment file
cat > docker-swarm/.env.production <<EOF
# Docker Registry
DOCKER_REGISTRY=yourdockerhub  # or your-vps-ip:5000
VERSION=latest
ENVIRONMENT=production
MODE=db

# Redis Passwords (CHANGE THESE!)
USER_REDIS_PASSWORD=$(openssl rand -base64 32)
WALLET_REDIS_PASSWORD=$(openssl rand -base64 32)
TRANSACTION_REDIS_PASSWORD=$(openssl rand -base64 32)
EOF
```

### Step 4: Transfer Files to VPS

From your local machine:

```bash
# Copy necessary files
scp -r 0-nginx root@your-vps-ip:/root/omni-server/
scp -r docker-swarm root@your-vps-ip:/root/omni-server/

# Or clone from Git (recommended)
# On VPS:
cd /root/omni-server
git clone https://github.com/yourusername/omni-server.git .
```

### Step 5: Build Images (If Using Local Registry)

If using a private registry on VPS:

```bash
# On your local machine
# Set registry to VPS
export DOCKER_REGISTRY=your-vps-ip:5000

# Build and push
./docker-swarm/local-build.sh
```

If using Docker Hub, build and push from CI/CD or local:

```bash
# Build each service
docker build -t yourdockerhub/user-service:latest ./1-users
docker push yourdockerhub/user-service:latest

docker build -t yourdockerhub/wallet-service:latest ./3-wallet
docker push yourdockerhub/wallet-service:latest

docker build -t yourdockerhub/notification-service:latest ./2-notification --target production
docker push yourdockerhub/notification-service:latest

docker build -t yourdockerhub/transaction-service:latest ./4-transactions
docker push yourdockerhub/transaction-service:latest

docker build -t yourdockerhub/fraud-detection-service:latest ./5-fraud-detection
docker push yourdockerhub/fraud-detection-service:latest
```

### Step 6: Deploy on VPS

SSH into VPS:

```bash
ssh root@your-vps-ip
cd /root/omni-server

# Load environment variables
export $(cat docker-swarm/.env.production | grep -v '^#' | xargs)

# Deploy stack
docker stack deploy -c docker-swarm/stack.yaml omni-server
```

### Step 7: Monitor Deployment

```bash
# Watch services start
watch docker stack services omni-server

# Check service logs
docker service logs -f omni-server_user-service
docker service logs -f omni-server_transaction-service
docker service logs -f omni-server_fraud-detection-service

# Check for errors
docker service ps omni-server_user-service --no-trunc
```

### Step 8: Configure Firewall

```bash
# Allow HTTP
ufw allow 80/tcp

# Allow HTTPS (if using SSL)
ufw allow 443/tcp

# Allow SSH
ufw allow 22/tcp

# Enable firewall
ufw enable
```

### Step 9: Test Production Deployment

```bash
# Test from another machine
curl http://your-vps-ip/health
curl http://your-vps-ip/api/users/health
curl http://your-vps-ip/api/wallets/health
curl http://your-vps-ip/api/transactions/health
curl http://your-vps-ip/api/fraud-detection/health
```

### Step 10: Update Deployment (Rolling Update)

When you have new code:

```bash
# Build new version
docker build -t yourdockerhub/user-service:v1.1.0 ./1-users
docker push yourdockerhub/user-service:v1.1.0

# On VPS, update service
docker service update \
  --image yourdockerhub/user-service:v1.1.0 \
  omni-server_user-service

# Or update entire stack
export VERSION=v1.1.0
docker stack deploy -c docker-swarm/stack.yaml omni-server
```

### Step 11: Backup and Restore

**Backup Redis Data:**

```bash
# Create backup directory
mkdir -p /backups

# Backup Redis volumes
docker run --rm \
  -v omni-server_user_redis_data:/data \
  -v /backups:/backup \
  alpine tar czf /backup/user-redis-$(date +%Y%m%d).tar.gz -C /data .

docker run --rm \
  -v omni-server_wallet_redis_data:/data \
  -v /backups:/backup \
  alpine tar czf /backup/wallet-redis-$(date +%Y%m%d).tar.gz -C /data .

docker run --rm \
  -v omni-server_transaction_redis_data:/data \
  -v /backups:/backup \
  alpine tar czf /backup/transaction-redis-$(date +%Y%m%d).tar.gz -C /data .
```

**Restore Redis Data:**

```bash
# Stop stack
docker stack rm omni-server

# Restore volume
docker run --rm \
  -v omni-server_user_redis_data:/data \
  -v /backups:/backup \
  alpine sh -c "cd /data && tar xzf /backup/user-redis-20251019.tar.gz"

# Restart stack
docker stack deploy -c docker-swarm/stack.yaml omni-server
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check service logs
docker service logs omni-server_user-service

# Check why replicas aren't running
docker service ps omni-server_user-service --no-trunc

# Inspect service
docker service inspect omni-server_user-service
```

### Service Keeps Restarting

```bash
# Check health check
docker service inspect omni-server_user-service --format='{{json .Spec.TaskTemplate.ContainerSpec.Healthcheck}}'

# View detailed logs
docker service logs --tail 100 omni-server_user-service

# Check resource limits
docker service inspect omni-server_user-service --format='{{json .Spec.TaskTemplate.Resources}}'
```

### Can't Connect to Service

```bash
# Check if service is running
docker stack services omni-server

# Test service directly (exec into container)
docker exec -it $(docker ps -q -f name=omni-server_user-service) sh

# Inside container, test service
wget -O- http://localhost:8080/api/users/health

# Check nginx config
docker service logs omni-server_nginx

# Test DNS resolution
docker exec -it $(docker ps -q -f name=omni-server_nginx) sh
nslookup user-service
```

### High Memory/CPU Usage

```bash
# Check resource usage
docker stats

# Scale down a service
docker service scale omni-server_user-service=1

# Update resource limits
docker service update \
  --limit-memory=512m \
  --limit-cpu=0.5 \
  omni-server_user-service
```

### Kafka Issues

```bash
# Check Kafka logs
docker service logs omni-server_broker

# Check if Kafka is healthy
docker exec -it $(docker ps -q -f name=omni-server_broker) \
  /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server broker:9092

# List topics
docker exec -it $(docker ps -q -f name=omni-server_broker) \
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server broker:9092 --list
```

### Redis Connection Issues

```bash
# Test Redis connection
docker exec -it $(docker ps -q -f name=omni-server_user-redis) \
  redis-cli -a your-password ping

# Check Redis data
docker exec -it $(docker ps -q -f name=omni-server_user-redis) \
  redis-cli -a your-password DBSIZE
```

### Clean Slate (Nuclear Option)

```bash
# WARNING: This removes everything!

# Remove stack
docker stack rm omni-server

# Wait for services to stop
sleep 30

# Remove all volumes
docker volume prune -f

# Remove all unused networks
docker network prune -f

# Redeploy
docker stack deploy -c docker-swarm/stack.yaml omni-server
```

---

## SSL/HTTPS Setup (Production)

### Using Let's Encrypt with Certbot

```bash
# Install certbot
apt-get install certbot

# Get certificate
certbot certonly --standalone -d yourdomain.com

# Certificates will be in /etc/letsencrypt/live/yourdomain.com/

# Update nginx config to use SSL (manual step required)

# Auto-renew
echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -
```

### Update Nginx for HTTPS

Edit `0-nginx/nginx-swarm.conf` to add SSL configuration, then redeploy.

---

## Monitoring and Logging

### View Logs in Real-Time

```bash
# All services
docker service ls --format "table {{.Name}}\t{{.Replicas}}"

# Specific service
docker service logs -f --tail 100 omni-server_user-service
```

### Export Logs

```bash
# Save logs to file
docker service logs omni-server_user-service > user-service.log
```

### Monitoring Tools (Optional)

Consider adding:
- Prometheus + Grafana for metrics
- ELK Stack for centralized logging
- Portainer for web-based management

---

## Performance Tuning

### Increase Replicas

```bash
docker service scale \
  omni-server_user-service=10 \
  omni-server_wallet-service=10 \
  omni-server_transaction-service=8
```

### Adjust Resources

```bash
docker service update \
  --limit-memory=1g \
  --limit-cpu=2 \
  --reserve-memory=512m \
  --reserve-cpu=1 \
  omni-server_transaction-service
```

---

## Next Steps

1. Set up CI/CD pipeline (GitHub Actions, GitLab CI)
2. Configure domain name and SSL
3. Set up monitoring and alerts
4. Implement log aggregation
5. Configure automated backups
6. Set up staging environment

For more details, see:
- [Transaction Flow Documentation](TRANSACTION_FLOW.md)
- [API Documentation](4-transactions/TRANSACTIONS_API.md)
