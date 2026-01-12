#!/bin/bash
# Complete deployment script - run this after SSH key is configured
# Usage: bash deploy-now.sh

set -e

SERVER="root@115.190.239.146"
REMOTE_DIR="/opt/lurus"
LOCAL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================="
echo "Lurus Switch Deployment"
echo "Server: 115.190.239.146"
echo "========================================="
echo ""

# Step 1: Verify SSH connection
echo "Step 1: Verifying SSH connection..."
if ssh -o BatchMode=yes -o ConnectTimeout=10 $SERVER "echo 'OK'" 2>/dev/null | grep -q "OK"; then
    echo "[OK] SSH connection verified"
else
    echo "[ERROR] SSH key not configured. Please run:"
    echo "cat ~/.ssh/id_rsa.pub | ssh $SERVER \"mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys\""
    echo "Password: GGsuperman1211"
    exit 1
fi

# Step 2: Check environment
echo ""
echo "Step 2: Checking server environment..."
ssh $SERVER "docker --version && docker compose version"

# Step 3: Create directories
echo ""
echo "Step 3: Creating directories..."
ssh $SERVER "mkdir -p $REMOTE_DIR/{deploy,data,logs,backup}"

# Step 4: Copy files
echo ""
echo "Step 4: Copying configuration files..."
cd "$LOCAL_DIR"
scp docker-compose.dev.yaml $SERVER:$REMOTE_DIR/
scp -r deploy $SERVER:$REMOTE_DIR/ 2>/dev/null || true
scp .env.production $SERVER:$REMOTE_DIR/.env 2>/dev/null || true

# Step 5: Verify files
echo ""
echo "Step 5: Verifying files..."
ssh $SERVER "ls -lh $REMOTE_DIR/docker-compose.dev.yaml $REMOTE_DIR/.env"

# Step 6: Pull Docker images
echo ""
echo "Step 6: Pulling Docker images (this may take 5-10 minutes)..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml pull"

# Step 7: Start infrastructure services
echo ""
echo "Step 7: Starting infrastructure services..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"

echo ""
echo "Waiting 30 seconds for services to initialize..."
for i in {30..1}; do
    printf "\rTime remaining: %2d seconds" $i
    sleep 1
done
echo ""

# Step 8: Check infrastructure
echo ""
echo "Step 8: Checking infrastructure services..."
ssh $SERVER "docker ps --filter 'name=lurus-' --format 'table {{.Names}}\t{{.Status}}'"

# Step 9: Start observability services
echo ""
echo "Step 9: Starting observability services..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"

echo "Waiting 10 seconds..."
sleep 10

# Step 10: Final status check
echo ""
echo "Step 10: Final deployment status..."
ssh $SERVER "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep lurus"

# Step 11: Initialize databases
echo ""
echo "Step 11: Checking databases..."
ssh $SERVER "docker exec lurus-postgres psql -U lurus -c '\l'" 2>/dev/null || echo "Database will be initialized on first connection"

echo ""
echo "========================================="
echo "Deployment Complete!"
echo "========================================="
echo ""
echo "Access URLs:"
echo "  Grafana:    http://115.190.239.146:3000 (admin/admin)"
echo "  Prometheus: http://115.190.239.146:9090"
echo "  Jaeger:     http://115.190.239.146:16686"
echo "  Consul:     http://115.190.239.146:8500"
echo ""
echo "Check logs:"
echo "  ssh $SERVER 'docker logs -f lurus-postgres'"
echo ""
echo "View all containers:"
echo "  ssh $SERVER 'docker ps'"
echo ""
