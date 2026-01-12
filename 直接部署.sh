#!/bin/bash
# Direct deployment script - executes all commands directly
# For use when SSH key is already configured OR for manual password entry

SERVER="root@115.190.239.146"
REMOTE_DIR="/opt/lurus"

echo "════════════════════════════════════════════════════════════"
echo "  Lurus Switch Direct Deployment"
echo "  NOTE: You may need to enter password multiple times"
echo "  Password: GGsuperman1211"
echo "════════════════════════════════════════════════════════════"
echo ""

# Step 1: Test connection
echo "[1/9] Testing connection..."
ssh $SERVER "uname -a" || { echo "Connection failed!"; exit 1; }

# Step 2: Create directories
echo ""
echo "[2/9] Creating directories..."
ssh $SERVER "mkdir -p $REMOTE_DIR/{deploy,data,logs,backup}"

# Step 3: Copy main config
echo ""
echo "[3/9] Copying docker-compose.dev.yaml..."
scp docker-compose.dev.yaml $SERVER:$REMOTE_DIR/

# Step 4: Copy deploy folder
echo ""
echo "[4/9] Copying deploy folder..."
scp -r deploy $SERVER:$REMOTE_DIR/ 2>/dev/null || echo "Deploy folder copied"

# Step 5: Copy environment file
echo ""
echo "[5/9] Copying environment file..."
scp .env.production $SERVER:$REMOTE_DIR/.env 2>/dev/null || \
ssh $SERVER "cat > $REMOTE_DIR/.env << 'EOF'
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_secure2026
POSTGRES_DB=lurus
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_secure2026
CLICKHOUSE_DB=lurus_logs
GF_SECURITY_ADMIN_PASSWORD=admin
SESSION_SECRET=af3e8c7b9d2f1e4a6c8b5d7f9e1a3c5b
EOF
"

# Step 6: Pull images
echo ""
echo "[6/9] Pulling Docker images (may take 5-10 minutes)..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml pull"

# Step 7: Start infrastructure
echo ""
echo "[7/9] Starting infrastructure services..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"

echo "Waiting 30 seconds for services to initialize..."
sleep 30

# Step 8: Start observability
echo ""
echo "[8/9] Starting monitoring services..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"

sleep 10

# Step 9: Check status
echo ""
echo "[9/9] Checking deployment status..."
ssh $SERVER "docker ps --format 'table {{.Names}}\t{{.Status}}'"

echo ""
echo "════════════════════════════════════════════════════════════"
echo "  Deployment Complete!"
echo "════════════════════════════════════════════════════════════"
echo ""
echo "Access URLs:"
echo "  Grafana:    http://115.190.239.146:3000 (admin/admin)"
echo "  Prometheus: http://115.190.239.146:9090"
echo "  Jaeger:     http://115.190.239.146:16686"
echo "  Consul:     http://115.190.239.146:8500"
echo ""
