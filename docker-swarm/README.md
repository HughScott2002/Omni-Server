# Docker Swarm Deployment on DigitalOcean

Complete setup for deploying Omni Server to Docker Swarm on DigitalOcean with automatic CI/CD from GitHub.

## 🎯 Overview

- **Development**: Work locally, push to GitHub
- **CI/CD**: GitHub Actions automatically builds and deploys
- **Production**: Docker Swarm on DigitalOcean VPS
- **Auto-healing**: Services restart automatically if they fail
- **Scaling**: Easy to scale services up/down
- **Cost**: ~$40-80/month (much cheaper than K8s!)

## 📋 Prerequisites

1. DigitalOcean account
2. GitHub repository
3. Local Docker installed
4. SSH key for server access

## 🚀 Setup Steps

### Step 1: Create DigitalOcean Droplets

Create 2-3 Ubuntu droplets:

**Recommended Setup** (~$72/month):
- 3x droplets: 4GB RAM, 2 vCPUs each ($24/month each)
- Choose region closest to your users
- Enable "Docker" from One-Click Apps

**Budget Setup** (~$36/month):
- 2x droplets: 2GB RAM, 1 vCPU each ($18/month each)

**DigitalOcean CLI Setup**:
```bash
# Install doctl
brew install doctl  # macOS
# or
snap install doctl  # Linux

# Authenticate
doctl auth init
```

**Create droplets via CLI** (optional):
```bash
# Create 3 droplets with Docker pre-installed
doctl compute droplet create \
  swarm-manager-1 \
  --size s-2vcpu-4gb \
  --image docker-20-04 \
  --region nyc3 \
  --ssh-keys $(doctl compute ssh-key list --format ID --no-header | head -1)

doctl compute droplet create \
  swarm-worker-1 swarm-worker-2 \
  --size s-2vcpu-4gb \
  --image docker-20-04 \
  --region nyc3 \
  --ssh-keys $(doctl compute ssh-key list --format ID --no-header | head -1)
```

### Step 2: Configure Firewall

Open these ports on your droplets:

**Manager Node**:
- TCP 22 (SSH)
- TCP 80 (HTTP)
- TCP 443 (HTTPS)
- TCP 2377 (Swarm management)
- TCP/UDP 7946 (Node communication)
- UDP 4789 (Overlay network)

**Worker Nodes**:
- TCP 22 (SSH)
- TCP 2377 (Swarm management)
- TCP/UDP 7946 (Node communication)
- UDP 4789 (Overlay network)

**Using DigitalOcean Firewall** (recommended):
```bash
# Create firewall
doctl compute firewall create \
  --name swarm-firewall \
  --inbound-rules "protocol:tcp,ports:22,sources:addresses:0.0.0.0/0,sources:addresses:::/0 protocol:tcp,ports:80,sources:addresses:0.0.0.0/0,sources:addresses:::/0 protocol:tcp,ports:443,sources:addresses:0.0.0.0/0,sources:addresses:::/0 protocol:tcp,ports:2377,sources:addresses:10.0.0.0/8 protocol:tcp,ports:7946,sources:addresses:10.0.0.0/8 protocol:udp,ports:7946,sources:addresses:10.0.0.0/8 protocol:udp,ports:4789,sources:addresses:10.0.0.0/8"
```

### Step 3: Initialize Docker Swarm

Get your droplet IPs:
```bash
doctl compute droplet list
```

Initialize the swarm:
```bash
# SSH into your manager node
ssh root@YOUR_MANAGER_IP

# Initialize swarm
docker swarm init --advertise-addr YOUR_MANAGER_IP

# You'll get a join token, save it!
```

Join worker nodes:
```bash
# SSH into each worker node
ssh root@YOUR_WORKER_IP

# Use the join token from above
docker swarm join --token SWMTKN-xxx YOUR_MANAGER_IP:2377
```

Or use the automated script:
```bash
chmod +x docker-swarm/setup-digitalocean.sh
./docker-swarm/setup-digitalocean.sh YOUR_MANAGER_IP YOUR_WORKER1_IP YOUR_WORKER2_IP
```

### Step 4: Set Up DigitalOcean Container Registry

```bash
# Create registry
doctl registry create omni-server

# Get registry name
doctl registry get

# Update docker-swarm/.env.production with your registry name:
# DOCKER_REGISTRY=registry.digitalocean.com/omni-server
```

### Step 5: Configure GitHub Secrets

Add these secrets in your GitHub repo (Settings → Secrets and variables → Actions):

1. **DIGITALOCEAN_ACCESS_TOKEN**
   - Get from: https://cloud.digitalocean.com/account/api/tokens
   - Click "Generate New Token"
   - Name: "GitHub Actions"
   - Enable both Read and Write scopes

2. **DIGITALOCEAN_REGISTRY_NAME**
   - Value: `omni-server` (or whatever you named your registry)

3. **SWARM_MANAGER_HOST**
   - Value: Your manager node IP (e.g., `192.168.1.10`)

4. **SSH_PRIVATE_KEY**
   - Your SSH private key to access the manager node
   - Get with: `cat ~/.ssh/id_rsa`

5. **USER_REDIS_PASSWORD**
   - A strong password for Redis
   - Generate with: `openssl rand -base64 32`

### Step 6: Test Locally First

Before deploying to production, test locally:

```bash
# Build images locally
./docker-swarm/local-build.sh

# Deploy to local swarm
./docker-swarm/deploy-local.sh

# Test
curl http://localhost/api/users

# Clean up
docker stack rm omni-server
```

### Step 7: Deploy to Production

**Option A: Push to GitHub (Recommended)**
```bash
git add .
git commit -m "Set up Docker Swarm deployment"
git push origin main
```

GitHub Actions will automatically:
1. Build the Docker image
2. Push to DigitalOcean Container Registry
3. Deploy to your Swarm cluster

**Option B: Manual Deploy**
```bash
# Build and push manually
doctl registry login
docker build -t registry.digitalocean.com/omni-server/user-service:latest ./1-users
docker push registry.digitalocean.com/omni-server/user-service:latest

# Deploy
ssh root@YOUR_MANAGER_IP
cd /root/omni-server
docker stack deploy -c docker-swarm/stack.yaml omni-server
```

## 📊 Managing Your Swarm

### View Services
```bash
ssh root@YOUR_MANAGER_IP
docker stack services omni-server
docker stack ps omni-server
```

### Scale Services
```bash
# Scale user-service to 5 replicas
docker service scale omni-server_user-service=5

# Scale nginx to 3 replicas
docker service scale omni-server_nginx=3
```

### View Logs
```bash
# Follow logs for user-service
docker service logs -f omni-server_user-service

# Last 100 lines
docker service logs --tail 100 omni-server_user-service
```

### Update Services
```bash
# Update user-service image
docker service update \
  --image registry.digitalocean.com/omni-server/user-service:latest \
  omni-server_user-service

# Update with zero downtime
docker service update \
  --update-parallelism 1 \
  --update-delay 10s \
  omni-server_user-service
```

### Remove Stack
```bash
docker stack rm omni-server
```

## 🔄 Workflow

**Daily Development**:
1. Make changes locally
2. Test with `docker-compose up` or local Swarm
3. Commit and push to GitHub
4. GitHub Actions deploys automatically
5. Monitor in DigitalOcean

**Scaling for Traffic**:
```bash
# Quick scale up
ssh root@YOUR_MANAGER_IP
docker service scale omni-server_user-service=10

# Or update stack.yaml and redeploy
```

## 💰 Cost Breakdown

**Recommended Setup** (~$72/month):
- 3x 4GB droplets @ $24/month = $72
- Container Registry: FREE (first 500MB)
- Bandwidth: 3TB included
- **Total: ~$72/month**

**Budget Setup** (~$36/month):
- 2x 2GB droplets @ $18/month = $36
- Container Registry: FREE
- **Total: ~$36/month**

Compare to GKE: ~$91/month minimum!

## 📈 Capacity Estimates

**Budget Setup** (2x 2GB nodes):
- ~500-1000 concurrent users
- Good for testing/staging

**Recommended Setup** (3x 4GB nodes):
- ~2000-3000 concurrent users
- Production ready

**Scale Up** (5x 8GB nodes ~$240/month):
- ~5000-10000 concurrent users

## 🔒 Security Checklist

- [ ] Change Redis password in .env.production (USER_REDIS_PASSWORD)
- [ ] Set up SSL/TLS with Let's Encrypt
- [ ] Enable DigitalOcean firewall
- [ ] Restrict SSH to your IP
- [ ] Enable automatic security updates
- [ ] Set up monitoring/alerts

## 🎓 Useful Commands

```bash
# Cluster info
docker node ls
docker info

# Service info
docker service ls
docker service ps omni-server_user-service
docker service inspect omni-server_user-service

# Network info
docker network ls
docker network inspect omni-server_omni-network

# Drain node for maintenance
docker node update --availability drain worker-1

# Bring node back
docker node update --availability active worker-1
```

## 🆘 Troubleshooting

**Services not starting?**
```bash
docker service ps omni-server_user-service --no-trunc
docker service logs omni-server_user-service
```

**Can't connect to service?**
```bash
# Check if service is running
docker service ls

# Check ingress network
docker network inspect ingress

# Test from manager node
curl http://localhost/api/users
```

**Registry authentication issues?**
```bash
# Re-login to registry
doctl registry login
```

## 🚀 Next Steps

- [ ] Set up monitoring (Prometheus/Grafana)
- [ ] Configure backups for Redis
- [ ] Set up SSL with Let's Encrypt
- [ ] Add more services
- [ ] Set up staging environment
- [ ] Configure log aggregation

## 📚 Additional Resources

- [Docker Swarm Docs](https://docs.docker.com/engine/swarm/)
- [DigitalOcean Container Registry](https://docs.digitalocean.com/products/container-registry/)
- [GitHub Actions Docs](https://docs.github.com/en/actions)
