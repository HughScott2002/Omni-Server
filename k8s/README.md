# Kubernetes Deployment for Omni Server

This directory contains Kubernetes manifests and scripts for deploying Omni Server to Google Kubernetes Engine (GKE).

## Features

- **Self-Healing**: Kubernetes automatically restarts failed containers
- **Auto-Scaling**: Horizontal Pod Autoscaler (HPA) scales services based on CPU/memory
- **High Availability**: Multiple replicas with health checks
- **CI/CD Ready**: GitHub Actions workflow for automated deployments

## Prerequisites

1. **Google Cloud Account** with billing enabled
2. **gcloud CLI** installed and configured
3. **kubectl** installed
4. **Docker** installed

## Quick Start

### 1. Initial Setup

```bash
# Login to GCP
gcloud auth login

# Set your project ID in the scripts
# Edit these files and replace YOUR_GCP_PROJECT_ID:
# - k8s/setup-gke.sh
# - k8s/build-and-push.sh
# - k8s/update-deployment.sh
# - k8s/user-service-deployment.yaml (line 22)

# Create the GKE cluster
chmod +x k8s/*.sh
./k8s/setup-gke.sh
```

### 2. Build and Push Docker Image

```bash
# Build and push to Google Container Registry
./k8s/build-and-push.sh
```

### 3. Deploy to GKE

```bash
# Deploy all services
./k8s/deploy.sh

# Wait for external IP (may take 2-3 minutes)
kubectl get service nginx -n omni-server --watch
```

### 4. Access Your Application

Once you have the EXTERNAL-IP, you can access your service:

```bash
curl http://EXTERNAL-IP/api/users
```

## Architecture

### Services

- **NGINX**: Load balancer and reverse proxy (2-5 replicas)
- **User Service**: Go backend service (2-10 replicas with autoscaling)
- **Redis**: Persistent storage (StatefulSet with PVC)

### Auto-Scaling Configuration

**User Service HPA**:
- Min replicas: 2
- Max replicas: 10
- Scale up when CPU > 70% or Memory > 80%

**NGINX HPA**:
- Min replicas: 2
- Max replicas: 5
- Scale up when CPU > 70%

### Health Checks

All services have:
- **Liveness Probes**: Restart unhealthy containers
- **Readiness Probes**: Stop sending traffic to unhealthy pods

## CI/CD with GitHub Actions

### Setup GitHub Actions

1. **Create a GCP Service Account**:
```bash
gcloud iam service-accounts create github-actions \
    --display-name="GitHub Actions"

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:github-actions@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/container.developer"

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:github-actions@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/storage.admin"

gcloud iam service-accounts keys create key.json \
    --iam-account=github-actions@YOUR_PROJECT_ID.iam.gserviceaccount.com
```

2. **Add GitHub Secrets**:
   - Go to your repo → Settings → Secrets and variables → Actions
   - Add these secrets:
     - `GCP_PROJECT_ID`: Your GCP project ID
     - `GCP_SA_KEY`: Contents of the `key.json` file

3. **Push to main branch** to trigger deployment

## Common Commands

```bash
# View all resources
kubectl get all -n omni-server

# Check pod logs
kubectl logs -f deployment/user-service -n omni-server
kubectl logs -f deployment/nginx -n omni-server

# Check HPA status
kubectl get hpa -n omni-server

# Scale manually
kubectl scale deployment user-service --replicas=5 -n omni-server

# Update deployment with new image
./k8s/update-deployment.sh

# Delete everything
kubectl delete namespace omni-server

# Delete the cluster
gcloud container clusters delete omni-server-cluster --region=us-central1
```

## Cost Optimization

**Estimated monthly cost** (us-central1):
- 3 x e2-medium nodes: ~$73/month
- Load Balancer: ~$18/month
- **Total: ~$91/month**

To reduce costs:
- Use smaller machine types (e2-small)
- Reduce minimum node count to 1-2
- Use preemptible nodes (not recommended for production)
- Delete cluster when not in use

## Monitoring

GKE includes built-in monitoring with Google Cloud Operations (formerly Stackdriver).

View metrics in Google Cloud Console:
1. Go to Kubernetes Engine → Clusters
2. Click on your cluster
3. Navigate to "Workloads" or "Services & Ingress"

## Troubleshooting

**Pods not starting?**
```bash
kubectl describe pod POD_NAME -n omni-server
kubectl logs POD_NAME -n omni-server
```

**Can't reach external IP?**
```bash
# Check if LoadBalancer has external IP
kubectl get service nginx -n omni-server

# Check firewall rules
gcloud compute firewall-rules list
```

**Database connection issues?**
```bash
# Check if Redis is running
kubectl get statefulset -n omni-server
kubectl logs statefulset/user-redis -n omni-server
```

## Security Notes

⚠️ **Important**: Before deploying to production:

1. Change the Redis password in `k8s/secret.yaml`
2. Use Google Secret Manager instead of K8s secrets
3. Enable network policies
4. Set up HTTPS with cert-manager
5. Implement proper RBAC

## Next Steps

- [ ] Set up monitoring and alerting
- [ ] Configure backup for Redis data
- [ ] Add more services (wallet, notification, transaction)
- [ ] Set up staging environment
- [ ] Implement blue/green deployments
- [ ] Add Ingress with SSL/TLS
