# HomeBoard — AWS Deployment & Cloud Configuration Guide

This guide details how to configure AWS Bedrock and deploy HomeBoard with automated HTTPS, zero-secret repository hygiene, and cost controls for the **AWS Builder Challenge**.

---

## 1. AWS Bedrock Access Setup (Prerequisite for AI Features)

### Step 1.1: Request Model Access in AWS Bedrock Console
Amazon Bedrock requires enabling foundation models once per AWS account:
1. Open the [AWS Bedrock Console](https://console.aws.amazon.com/bedrock/home?region=us-east-1).
2. Select **us-east-1 (N. Virginia)** in the top-right region selector (Bedrock models have the broadest availability in `us-east-1`).
3. In the left sidebar, click **Model access** (under the *Bedrock configurations* section).
4. Click **Modify model access** or **Enable specific models**.
5. Select:
   - **Anthropic**: `Claude 3 Haiku` (model ID: `anthropic.claude-3-haiku-20240307-v1:0`)
   - *(Optional)* **Amazon**: `Amazon Nova Micro` / `Lite`
6. Click **Next** and **Submit**. Approval for Claude 3 Haiku is instant.

### Step 1.2: Grant IAM Permissions to User `myself`
Your AWS CLI identity is `arn:aws:iam::998850986995:user/myself`. To allow the CLI and backend to invoke Bedrock models:
1. Open the [IAM Console Users Page](https://console.aws.amazon.com/iam/home#/users/myself).
2. Click **Add permissions** -> **Attach policies directly**.
3. Search for and select:
   - `AmazonBedrockFullAccess` *(or custom policy below)*
   - `AWSAppRunnerFullAccess` *(if deploying via AWS App Runner)*
   - `AmazonEC2ContainerRegistryFullAccess` *(for ECR image uploads)*
4. Click **Next** -> **Add permissions**.

#### Minimal IAM Policy for Bedrock (Least Privilege)
If you prefer least privilege instead of full access:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BedrockInvokeModelAccess",
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:ListFoundationModels"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## 2. Testing Bedrock from CLI
Once permissions are attached, verify Bedrock via your terminal:
```bash
aws bedrock list-foundation-models --region us-east-1 --by-provider anthropic --query "modelSummaries[*].modelId"
```
You should see:
```json
[
    "anthropic.claude-3-haiku-20240307-v1:0",
    "anthropic.claude-3-sonnet-20240229-v1:0"
]
```

---

## 3. Production Deployment Options

### Option A: AWS App Runner (Recommended — 1-Click Serverless Containers with Free HTTPS)
AWS App Runner automatically builds and runs your container with built-in HTTPS:
1. Go to [AWS App Runner Console](https://console.aws.amazon.com/apprunner/home?region=us-east-1).
2. Click **Create service**.
3. Choose **Source code repository** -> Connect your GitHub account -> Select repository `kuldeep-poonia/homeboard` (Branch: `main`).
4. Configuration:
   - **Runtime**: `Go 1` (or **Container image** if using Dockerfile).
   - **Port**: `8080`.
   - **Build command**: `cd backend && go build -o bin/homeboard ./cmd/server`
   - **Start command**: `./backend/bin/homeboard`
5. Environment Variables:
   - `APP_ENV`: `production`
   - `BEDROCK_ENABLED`: `true`
   - `AWS_REGION`: `us-east-1`
   - `BEDROCK_MODEL_ID`: `anthropic.claude-3-haiku-20240307-v1:0`
6. Click **Deploy**. App Runner will provide a live HTTPS URL (e.g., `https://xyz123.us-east-1.awsapprunner.com`).

---

### Option B: Docker Container Deployment
If deploying on an EC2 instance, Lightsail, or ECS:
```bash
# Build and run with Docker Compose
docker compose up -d --build

# View logs
docker compose logs -f
```

---

## 4. Cost Control & Budget Alerts (AWS Well-Architected)
To ensure zero unexpected costs:
1. In the AWS Console, open **AWS Budgets**.
2. Click **Create budget** -> **Cost budget**.
3. Set budget amount: `$5.00 USD / month`.
4. Add email alert threshold at `80% ($4.00)`.
