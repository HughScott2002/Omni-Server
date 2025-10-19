#!/bin/bash
set -e

# Configuration - UPDATE THESE VALUES
PROJECT_ID="YOUR_GCP_PROJECT_ID"
SERVICE_NAME="user-service"
VERSION="${1:-latest}"

echo "================================================"
echo "Building and pushing $SERVICE_NAME to GCR"
echo "================================================"

# Configure Docker to use gcloud as a credential helper
gcloud auth configure-docker

# Build the Docker image
echo "Building Docker image..."
docker build -t gcr.io/$PROJECT_ID/$SERVICE_NAME:$VERSION ./1-users

# Tag as latest as well
docker tag gcr.io/$PROJECT_ID/$SERVICE_NAME:$VERSION gcr.io/$PROJECT_ID/$SERVICE_NAME:latest

# Push to Google Container Registry
echo "Pushing to GCR..."
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:$VERSION
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:latest

echo "================================================"
echo "Build and push complete!"
echo "Image: gcr.io/$PROJECT_ID/$SERVICE_NAME:$VERSION"
echo "================================================"
