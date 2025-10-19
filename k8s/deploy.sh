#!/bin/bash
set -e

echo "================================================"
echo "Deploying Omni Server to GKE"
echo "================================================"

# Apply manifests in order
echo "Creating namespace..."
kubectl apply -f k8s/namespace.yaml

echo "Creating ConfigMap and Secrets..."
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml

echo "Deploying Redis..."
kubectl apply -f k8s/redis-statefulset.yaml

echo "Deploying User Service..."
kubectl apply -f k8s/user-service-deployment.yaml

echo "Deploying NGINX..."
kubectl apply -f k8s/nginx-deployment.yaml

echo "Setting up Horizontal Pod Autoscaling..."
kubectl apply -f k8s/hpa.yaml

echo ""
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/user-service deployment/nginx -n omni-server

echo ""
echo "================================================"
echo "Deployment complete!"
echo "================================================"
echo ""

# Get the external IP
echo "Getting LoadBalancer external IP (this may take a few minutes)..."
kubectl get service nginx -n omni-server

echo ""
echo "Run this command to watch for the EXTERNAL-IP:"
echo "  kubectl get service nginx -n omni-server --watch"
echo ""
echo "Check pod status:"
echo "  kubectl get pods -n omni-server"
echo ""
echo "View logs:"
echo "  kubectl logs -f deployment/user-service -n omni-server"
