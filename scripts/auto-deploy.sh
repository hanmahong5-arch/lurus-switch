#!/bin/bash
# Fully automated deployment script using password
# Requires: sshpass (install with: apt-cyg install sshpass)

set -e

SERVER_IP="115.190.239.146"
SERVER_USER="root"
SERVER_PASSWORD="GGsuperman1211"
REMOTE_DIR="/opt/lurus"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to execute SSH command with password
ssh_exec() {
    local cmd="$1"
    sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP} "$cmd"
}

# Function to copy files with password
scp_copy() {
    local src="$1"
    local dest="$2"
    sshpass -p "$SERVER_PASSWORD" scp -o StrictHostKeyChecking=no -r "$src" ${SERVER_USER}@${SERVER_IP}:"$dest"
}

# Check if sshpass is installed
if ! command -v sshpass &> /dev/null; then
    echo -e "${YELLOW}sshpass not found. Installing...${NC}"
    echo "Please run: apt-cyg install sshpass"
    echo "Or use the manual deployment script: bash scripts/quick-deploy.sh"
    exit 1
fi

echo -e "${GREEN}=== Lurus Switch Automated Deployment ===${NC}"

# Step 1: Setup SSH key (with password)
echo -e "${YELLOW}Step 1: Setting up SSH key...${NC}"
cat ~/.ssh/id_rsa.pub | sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP} "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
echo -e "${GREEN}✓ SSH key configured${NC}"

# Test password-less connection
if ssh -o BatchMode=yes -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_IP} "echo OK" 2>/dev/null | grep -q "OK"; then
    echo -e "${GREEN}✓ SSH key authentication working${NC}"
    # From now on, we can use regular ssh without password
    USE_PASSWORD=false
else
    echo -e "${YELLOW}Using password authentication${NC}"
    USE_PASSWORD=true
fi

# Step 2: Check environment
echo ""
echo -e "${YELLOW}Step 2: Checking environment...${NC}"
if [ "$USE_PASSWORD" = true ]; then
    ssh_exec "uname -a && docker --version"
else
    ssh ${SERVER_USER}@${SERVER_IP} "uname -a && docker --version"
fi

# Step 3: Create directories
echo ""
echo -e "${YELLOW}Step 3: Creating directories...${NC}"
if [ "$USE_PASSWORD" = true ]; then
    ssh_exec "mkdir -p ${REMOTE_DIR}/{deploy,data,logs,backup}"
else
    ssh ${SERVER_USER}@${SERVER_IP} "mkdir -p ${REMOTE_DIR}/{deploy,data,logs,backup}"
fi

# Step 4: Copy files
echo ""
echo -e "${YELLOW}Step 4: Copying files...${NC}"
cd "$(dirname "$0")/.."
if [ "$USE_PASSWORD" = true ]; then
    scp_copy "docker-compose.dev.yaml" "${REMOTE_DIR}/"
    scp_copy "deploy" "${REMOTE_DIR}/"
else
    scp docker-compose.dev.yaml ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/
    scp -r deploy ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/
fi

# Step 5: Start services
echo ""
echo -e "${YELLOW}Step 5: Starting infrastructure...${NC}"
if [ "$USE_PASSWORD" = true ]; then
    ssh_exec "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse"
else
    ssh ${SERVER_USER}@${SERVER_IP} "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse"
fi

echo "Waiting 30 seconds..."
sleep 30

# Step 6: Check status
echo ""
echo -e "${YELLOW}Step 6: Checking status...${NC}"
if [ "$USE_PASSWORD" = true ]; then
    ssh_exec "docker ps"
else
    ssh ${SERVER_USER}@${SERVER_IP} "docker ps"
fi

echo ""
echo -e "${GREEN}=== Deployment Complete! ===${NC}"
echo "Access: ssh ${SERVER_USER}@${SERVER_IP}"
