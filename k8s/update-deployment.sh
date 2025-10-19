#!/bin/bash
set -e

# Quick script to rebuild, push, and update deployment
PROJECT_ID="YOUR_GCP_PROJECT_ID"
SERVICE_NAME="user-service"
VERSION=$(date +%Y%m%d-%H%M%S)

echo "Building and deploying version: $VERSION"

# Build and push
./k8s/build-and-push.sh $VERSION

# Update the deployment with new image
kubectl set image deployment/user-service \
  user-service=gcr.io/$PROJECT_ID/$SERVICE_NAME:$VERSION \
  -n omni-server

# Watch the rollout
kubectl rollout status deployment/user-service -n omni-server

echo "Deployment updated successfully!"
