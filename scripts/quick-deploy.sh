#!/bin/bash
# Quick deployment script for Lurus Switch
# This script will guide you through the deployment process

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SERVER_IP="115.190.239.146"
SERVER_USER="root"
SERVER_PASSWORD="GGsuperman1211"
REMOTE_DIR="/opt/lurus"

echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}Lurus Switch Quick Deployment${NC}"
echo -e "${GREEN}================================${NC}"
echo ""

# Step 1: Setup SSH Key
echo -e "${YELLOW}Step 1: Setting up SSH key authentication${NC}"
echo ""
echo "Checking if SSH key is already configured..."

if ssh -o BatchMode=yes -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_IP} "echo OK" 2>/dev/null | grep -q "OK"; then
    echo -e "${GREEN}✓ SSH key already configured!${NC}"
else
    echo -e "${YELLOW}SSH key not configured. Please run this command:${NC}"
    echo ""
    echo -e "${GREEN}cat ~/.ssh/id_rsa.pub | ssh ${SERVER_USER}@${SERVER_IP} \"mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo 'SSH configured!'\"${NC}"
    echo ""
    echo -e "${YELLOW}Password: ${GREEN}${SERVER_PASSWORD}${NC}"
    echo ""
    echo "After running the command, press Enter to continue..."
    read

    # Verify
    if ssh -o BatchMode=yes -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_IP} "echo OK" 2>/dev/null | grep -q "OK"; then
        echo -e "${GREEN}✓ SSH key configured successfully!${NC}"
    else
        echo -e "${RED}✗ SSH key configuration failed. Please check and try again.${NC}"
        exit 1
    fi
fi

# Step 2: Check environment
echo ""
echo -e "${YELLOW}Step 2: Checking server environment...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "
    echo '=== System ===' &&
    uname -a &&
    echo '' &&
    echo '=== Docker ===' &&
    docker --version &&
    docker compose version &&
    echo '' &&
    echo '=== Resources ===' &&
    echo 'Memory:' &&
    free -h | grep Mem &&
    echo 'Disk:' &&
    df -h / | tail -1
"

echo ""
echo "Environment check complete. Continue with deployment? (y/n)"
read -r response
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled."
    exit 0
fi

# Step 3: Prepare directories
echo ""
echo -e "${YELLOW}Step 3: Preparing deployment directories...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "mkdir -p ${REMOTE_DIR}/{deploy,data,logs,backup}"

# Step 4: Copy files
echo ""
echo -e "${YELLOW}Step 4: Copying configuration files...${NC}"
cd "$(dirname "$0")/.."

scp docker-compose.dev.yaml ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/
scp -r deploy ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/ 2>/dev/null || echo "Deploy directory already exists"
[ -f prometheus.yml ] && scp prometheus.yml ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/

echo -e "${GREEN}✓ Files copied${NC}"

# Step 5: Create .env file
echo ""
echo -e "${YELLOW}Step 5: Creating environment configuration...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "cat > ${REMOTE_DIR}/.env << 'EOF'
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_$(openssl rand -hex 8)
POSTGRES_DB=lurus
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_$(openssl rand -hex 8)
CLICKHOUSE_DB=lurus_logs
GF_SECURITY_ADMIN_PASSWORD=admin
SESSION_SECRET=$(openssl rand -hex 32)
EOF
"

echo -e "${GREEN}✓ Environment configured${NC}"

# Step 6: Pull images
echo ""
echo -e "${YELLOW}Step 6: Pulling Docker images (this may take a while)...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml pull"

echo -e "${GREEN}✓ Images pulled${NC}"

# Step 7: Start infrastructure
echo ""
echo -e "${YELLOW}Step 7: Starting infrastructure services...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"

echo ""
echo "Waiting 25 seconds for services to initialize..."
for i in {25..1}; do
    echo -ne "\rTime remaining: $i seconds "
    sleep 1
done
echo ""

# Step 8: Start observability
echo ""
echo -e "${YELLOW}Step 8: Starting observability services...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"

echo "Waiting 10 seconds..."
sleep 10

# Step 9: Check status
echo ""
echo -e "${YELLOW}Step 9: Checking deployment status...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

# Summary
echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo "Services running on ${SERVER_IP}:"
echo ""
echo "  Grafana:    http://${SERVER_IP}:3000 (admin/admin)"
echo "  Prometheus: http://${SERVER_IP}:9090"
echo "  Jaeger:     http://${SERVER_IP}:16686"
echo "  Consul:     http://${SERVER_IP}:8500"
echo ""
echo "Database connections:"
echo "  PostgreSQL: ${SERVER_IP}:5432"
echo "  Redis:      ${SERVER_IP}:6379"
echo "  NATS:       ${SERVER_IP}:4222"
echo "  ClickHouse: ${SERVER_IP}:8123 (HTTP), ${SERVER_IP}:9000 (Native)"
echo ""
echo "To view logs:"
echo "  ssh ${SERVER_USER}@${SERVER_IP} \"docker logs -f <container-name>\""
echo ""
echo "To deploy microservices, you need to build the images first."
echo "See DEPLOYMENT-STEPS.md for details."
echo ""
