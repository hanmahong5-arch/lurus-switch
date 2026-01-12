#!/bin/bash
# Final one-command deployment script
# This script will set up SSH and deploy everything

set -e

SERVER="root@115.190.239.146"
PASS="GGsuperman1211"
REMOTE_DIR="/opt/lurus"

echo "=== Lurus Switch Final Deployment ==="
echo ""

# Step 0: Setup SSH key non-interactively
echo "Step 0: Setting up SSH key..."

# Check if already configured
if ssh -o BatchMode=yes -o ConnectTimeout=5 $SERVER "echo OK" 2>/dev/null | grep -q "OK"; then
    echo "[OK] SSH key already configured"
else
    echo "Configuring SSH key (you may need to enter password once)..."

    # Method: Create a temporary expect-like script
    cat > /tmp/ssh-setup.sh << 'EOFSETUP'
#!/bin/bash
SERVER="root@115.190.239.146"
PUBKEY=$(cat ~/.ssh/id_rsa.pub)

# Use SSH with StrictHostKeyChecking=no for first-time connection
ssh -o StrictHostKeyChecking=no $SERVER << EOFSSH
mkdir -p ~/.ssh
chmod 700 ~/.ssh
echo "$PUBKEY" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
echo "SSH key configured!"
EOFSSH
EOFSETUP

    chmod +x /tmp/ssh-setup.sh

    echo "Please run: bash /tmp/ssh-setup.sh"
    echo "Password: $PASS"
    echo ""
    echo "Or run this single command:"
    echo "----------------------------------------"
    echo "cat ~/.ssh/id_rsa.pub | ssh root@115.190.239.146 \"mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys\""
    echo "----------------------------------------"
    echo ""
    echo "After setting up SSH key, run this script again."
    exit 0
fi

echo ""
echo "Step 1: Checking server..."
ssh $SERVER "echo 'Server:' && uname -a && echo '' && echo 'Docker:' && docker --version"

echo ""
echo "Step 2: Creating directories..."
ssh $SERVER "mkdir -p $REMOTE_DIR/{deploy,data,logs,backup}"

echo ""
echo "Step 3: Copying files..."
cd "$(dirname "$0")/.."
scp docker-compose.dev.yaml $SERVER:$REMOTE_DIR/
scp -r deploy $SERVER:$REMOTE_DIR/ 2>/dev/null || true

echo ""
echo "Step 4: Creating environment file..."
ssh $SERVER "cat > $REMOTE_DIR/.env << 'EOF'
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_secure123
POSTGRES_DB=lurus
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_secure123
CLICKHOUSE_DB=lurus_logs
GF_SECURITY_ADMIN_PASSWORD=admin
EOF
"

echo ""
echo "Step 5: Pulling Docker images..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml pull"

echo ""
echo "Step 6: Starting infrastructure services..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"

echo "Waiting 30 seconds for services to initialize..."
sleep 30

echo ""
echo "Step 7: Starting observability services..."
ssh $SERVER "cd $REMOTE_DIR && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"

sleep 10

echo ""
echo "Step 8: Checking deployment status..."
ssh $SERVER "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

echo ""
echo "=== Deployment Complete! ==="
echo ""
echo "Access services at:"
echo "  Grafana:    http://115.190.239.146:3000 (admin/admin)"
echo "  Prometheus: http://115.190.239.146:9090"
echo "  Jaeger:     http://115.190.239.146:16686"
echo "  Consul:     http://115.190.239.146:8500"
echo ""
echo "View logs: ssh $SERVER 'docker logs -f <container-name>'"
echo ""
