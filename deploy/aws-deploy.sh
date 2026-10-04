#!/bin/bash
set -euo pipefail

# HomeBoard AWS Deployment Script
# Deploys HomeBoard container service to AWS App Runner with HTTPS

SERVICE_NAME="homeboard-server"
REGION="${AWS_REGION:-us-east-1}"

echo "=========================================="
echo " HomeBoard AWS Production Deployment "
echo " Region: ${REGION}"
echo "=========================================="

# Check AWS CLI
if ! command -v aws &> /dev/null; then
    echo "Error: AWS CLI is not installed or not in PATH."
    exit 1
fi

echo "Verifying AWS Caller Identity..."
aws sts get-caller-identity --output table

ACCOUNT_ID=$(aws sts get-caller-identity --query "Account" --output text)
ECR_REPO_NAME="homeboard"
IMAGE_TAG="latest"
ECR_URI="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/${ECR_REPO_NAME}:${IMAGE_TAG}"

echo "1. Checking or creating ECR repository..."
aws ecr describe-repositories --repository-names "${ECR_REPO_NAME}" --region "${REGION}" &>/dev/null || \
    aws ecr create-repository --repository-name "${ECR_REPO_NAME}" --region "${REGION}" --image-scanning-configuration scanOnPush=true

echo "2. Authenticating Docker with Amazon ECR..."
aws ecr get-login-password --region "${REGION}" | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"

echo "3. Building container image..."
docker build -t "${ECR_URI}" -f Dockerfile .

echo "4. Pushing image to Amazon ECR..."
docker push "${ECR_URI}"

echo "5. Deploying to AWS App Runner..."
echo "Deployment image ready: ${ECR_URI}"
echo "You can now link this image in AWS App Runner or ECS Fargate."
