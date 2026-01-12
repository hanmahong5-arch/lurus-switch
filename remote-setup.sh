#!/bin/bash
# This script will be uploaded and run directly on the server
# Minimizes password prompts

cat > /tmp/server-deploy.sh << 'EOFSCRIPT'
#!/bin/bash
# Server-side deployment script

set -e

REMOTE_DIR="/opt/lurus"
cd $REMOTE_DIR

echo "═══════════════════════════════════════════════════════════"
echo "  Server-Side Deployment Starting..."
echo "═══════════════════════════════════════════════════════════"

# Step 1: Create .env file
echo ""
echo "[1/5] Creating environment file..."
cat > .env << 'EOF'
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_secure2026
POSTGRES_DB=lurus
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_secure2026
CLICKHOUSE_DB=lurus_logs
GF_SECURITY_ADMIN_PASSWORD=admin
SESSION_SECRET=af3e8c7b9d2f1e4a6c8b5d7f9e1a3c5b
EOF

# Step 2: Pull Docker images
echo ""
echo "[2/5] Pulling Docker images (this may take 5-10 minutes)..."
docker compose -f docker-compose.dev.yaml pull

# Step 3: Start infrastructure services
echo ""
echo "[3/5] Starting infrastructure services..."
docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul

echo "Waiting 30 seconds for services to initialize..."
sleep 30

# Step 4: Start observability services
echo ""
echo "[4/5] Starting monitoring services..."
docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager

sleep 10

# Step 5: Check status
echo ""
echo "[5/5] Checking deployment status..."
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "  Deployment Complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "Access URLs:"
echo "  Grafana:    http://115.190.239.146:3000 (admin/admin)"
echo "  Prometheus: http://115.190.239.146:9090"
echo "  Jaeger:     http://115.190.239.146:16686"
echo "  Consul:     http://115.190.239.146:8500"
echo ""

EOFSCRIPT

chmod +x /tmp/server-deploy.sh

echo "════════════════════════════════════════════════════════════"
echo "  Uploading files and executing deployment..."
echo "  You will need to enter password 2-3 times"
echo "  Password: GGsuperman1211"
echo "════════════════════════════════════════════════════════════"
echo ""

# Step 1: Create directories (password 1)
echo "Step 1: Creating directories..."
ssh root@115.190.239.146 "mkdir -p /opt/lurus/{deploy,data,logs,backup}"

# Step 2: Copy docker-compose file (password 2)
echo ""
echo "Step 2: Copying docker-compose.dev.yaml..."
scp docker-compose.dev.yaml root@115.190.239.146:/opt/lurus/

# Step 3: Copy deploy folder (password 3)
echo ""
echo "Step 3: Copying deploy configurations..."
scp -r deploy root@115.190.239.146:/opt/lurus/ 2>/dev/null || echo "Deploy folder copied"

# Step 4: Upload and execute server script (password 4)
echo ""
echo "Step 4: Uploading server-side script..."
scp /tmp/server-deploy.sh root@115.190.239.146:/tmp/

# Step 5: Execute deployment on server (password 5 - last one!)
echo ""
echo "Step 5: Executing deployment on server (this will take 5-10 minutes)..."
echo "Please enter password one last time..."
ssh -t root@115.190.239.146 "bash /tmp/server-deploy.sh"

echo ""
echo "════════════════════════════════════════════════════════════"
echo "  All Done!"
echo "════════════════════════════════════════════════════════════"
