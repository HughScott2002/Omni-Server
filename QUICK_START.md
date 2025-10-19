# Quick Start Guide

## Local Development (Fastest)

```bash
# 1. Start services
docker-compose up -d

# 2. Check status
docker-compose ps

# 3. Test
curl http://localhost/api/users/health
curl http://localhost/api/transactions/health
curl http://localhost/api/fraud-detection/health

# 4. View logs
docker-compose logs -f

# 5. Stop
docker-compose down
```

## Local Docker Swarm Testing

```bash
# 1. Initialize swarm
docker swarm init

# 2. Build images
chmod +x docker-swarm/*.sh
./docker-swarm/local-build.sh

# 3. Deploy
./docker-swarm/deploy-local.sh

# 4. Monitor
docker stack services omni-server
docker service logs -f omni-server_transaction-service

# 5. Test
curl http://localhost/health
curl http://localhost/api/transactions/health

# 6. Remove
docker stack rm omni-server
```

## VPS Production Deployment

```bash
# === ON LOCAL MACHINE ===

# 1. Build and push images
docker login
docker build -t yourdockerhub/user-service:latest ./1-users
docker push yourdockerhub/user-service:latest
# (repeat for all services: wallet, notification, transaction, fraud-detection)

# === ON VPS ===

# 2. SSH to VPS
ssh root@your-vps-ip

# 3. Install Docker (if needed)
curl -fsSL https://get.docker.com | sh

# 4. Initialize swarm
docker swarm init

# 5. Clone repo or copy files
git clone https://your-repo.git omni-server
cd omni-server

# 6. Configure environment
nano docker-swarm/.env.production
# Set DOCKER_REGISTRY=yourdockerhub
# Set strong passwords

# 7. Deploy
export $(cat docker-swarm/.env.production | xargs)
docker stack deploy -c docker-swarm/stack.yaml omni-server

# 8. Monitor
watch docker stack services omni-server
docker service logs -f omni-server_user-service

# 9. Test
curl http://your-vps-ip/api/users/health
```

## Common Issues

### Only user-service deploys
**Problem:** Stack files were incomplete
**Solution:** Updated stack.yaml and stack-local.yaml now include all services

### Services won't start
```bash
# Check logs
docker service logs omni-server_SERVICE_NAME

# Check why replicas failed
docker service ps omni-server_SERVICE_NAME --no-trunc
```

### Can't connect to services
```bash
# Check nginx is running
docker service ps omni-server_nginx

# Check service exists
docker stack services omni-server
```

## Service Architecture

```
Client
  ↓
Nginx (Port 80)
  ↓
┌─────────┬──────────┬──────────────┬─────────────┬─────────────────┐
│  Users  │ Wallets  │Notifications │Transactions │ Fraud Detection │
│  :8080  │  :8080   │    :8080     │    :8083    │      :8085      │
└─────────┴──────────┴──────────────┴─────────────┴─────────────────┘
     ↓         ↓            ↓              ↓
┌─────────┬──────────┬──────────────┬─────────────┐
│User     │Wallet    │Notification  │Transaction  │
│Redis    │Redis     │Redis         │Redis        │
└─────────┴──────────┴──────────────┴─────────────┘
           ↓
     Kafka Broker
```

## All Services Included

✅ Nginx (Load Balancer)
✅ Kafka (Message Broker)
✅ User Service + Redis
✅ Wallet Service + Redis
✅ Notification Service + Redis
✅ Transaction Service + Redis
✅ Fraud Detection Service (NEW!)

## Transaction Flow with Risk Scoring

1. Client sends transfer request → Transaction Service
2. Transaction Service validates request
3. Transaction Service calls Fraud Detection → Risk Assessment
4. If approved: Process transaction
5. If declined: Return error
6. Publish events to Kafka
7. Notification Service sends notifications

See [TRANSACTION_FLOW.md](TRANSACTION_FLOW.md) for detailed flow.

## Full Documentation

- **Deployment:** [DEPLOYMENT.md](DEPLOYMENT.md) - Complete deployment guide
- **Transaction Flow:** [TRANSACTION_FLOW.md](TRANSACTION_FLOW.md) - Transaction processing details
- **API Docs:** [4-transactions/TRANSACTIONS_API.md](4-transactions/TRANSACTIONS_API.md) - API reference
