#!/bin/bash
set -e

# Configuration - UPDATE THESE VALUES
PROJECT_ID="YOUR_GCP_PROJECT_ID"
CLUSTER_NAME="omni-server-cluster"
REGION="us-central1"
MACHINE_TYPE="e2-medium"
NUM_NODES="3"

echo "================================================"
echo "Setting up GKE cluster for Omni Server"
echo "================================================"

# Set the project
echo "Setting GCP project to: $PROJECT_ID"
gcloud config set project $PROJECT_ID

# Enable required APIs
echo "Enabling required GCP APIs..."
gcloud services enable container.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable cloudbuild.googleapis.com

# Create GKE cluster with autoscaling
echo "Creating GKE cluster: $CLUSTER_NAME"
gcloud container clusters create $CLUSTER_NAME \
  --region=$REGION \
  --machine-type=$MACHINE_TYPE \
  --num-nodes=$NUM_NODES \
  --enable-autoscaling \
  --min-nodes=2 \
  --max-nodes=10 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-stackdriver-kubernetes \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver

# Get cluster credentials
echo "Getting cluster credentials..."
gcloud container clusters get-credentials $CLUSTER_NAME --region=$REGION

# Verify connection
echo "Verifying cluster connection..."
kubectl cluster-info

echo "================================================"
echo "GKE cluster setup complete!"
echo "================================================"
echo ""
echo "Next steps:"
echo "1. Update k8s/user-service-deployment.yaml with your project ID"
echo "2. Build and push your Docker image: ./k8s/build-and-push.sh"
echo "3. Deploy to cluster: ./k8s/deploy.sh"
