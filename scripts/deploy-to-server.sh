#!/bin/bash
# Deployment script for Lurus Switch microservices
# Target server: 115.190.239.146

set -e

SERVER_IP="115.190.239.146"
SERVER_USER="root"
REMOTE_DIR="/opt/lurus"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Color output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}>>> $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}>>> $1${NC}"
}

log_error() {
    echo -e "${RED}>>> ERROR: $1${NC}"
}

remote_exec() {
    local cmd="$1"
    local desc="$2"
    log_info "$desc"
    ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP} "$cmd"
}

remote_copy() {
    local src="$1"
    local dest="$2"
    local desc="$3"
    log_info "$desc"
    scp -o StrictHostKeyChecking=no -r "$src" ${SERVER_USER}@${SERVER_IP}:"$dest"
}

# Step 1: Check SSH connection
log_info "Step 1: Checking SSH connection..."
if ! ssh -o BatchMode=yes -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_IP} "echo 'Connection OK'" 2>/dev/null; then
    log_warn "SSH key authentication not configured."
    log_warn "Setting up SSH key..."

    echo ""
    echo "Please enter the server password when prompted: GGsuperman1211"
    echo ""

    # Copy SSH public key
    cat ~/.ssh/id_rsa.pub | ssh ${SERVER_USER}@${SERVER_IP} "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

    log_info "SSH key configured. Testing connection..."
    remote_exec "echo 'SSH working!'" "Test SSH"
fi

# Step 2: Check server environment
log_info "Step 2: Checking server environment..."
remote_exec "uname -a" "Check OS"
remote_exec "cat /etc/os-release | head -5" "Check Linux distribution"
remote_exec "docker --version" "Check Docker"
remote_exec "docker compose version 2>&1 || docker-compose --version" "Check Docker Compose"

# Step 3: Check system resources
log_info "Step 3: Checking system resources..."
remote_exec "free -h | grep -E 'Mem|Swap'" "Check memory"
remote_exec "df -h / | tail -1" "Check disk space"

# Step 4: Check existing services
log_info "Step 4: Checking existing Docker containers..."
remote_exec "docker ps -a --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' 2>&1 || echo 'No containers found'" "List containers"

# Step 5: Create directories
log_info "Step 5: Creating deployment directories..."
remote_exec "mkdir -p ${REMOTE_DIR}/{deploy,data,logs,backup}" "Create directories"

# Step 6: Copy configuration files
log_info "Step 6: Copying configuration files..."
cd "$PROJECT_DIR"
remote_copy "docker-compose.dev.yaml" "${REMOTE_DIR}/" "Copy Docker Compose config"
remote_copy "deploy" "${REMOTE_DIR}/" "Copy deployment configs"

# Step 7: Create .env file
log_info "Step 7: Creating .env file..."
cat > /tmp/lurus.env << 'EOF'
# PostgreSQL
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_$(openssl rand -hex 16)
POSTGRES_DB=lurus

# Redis
REDIS_PASSWORD=$(openssl rand -hex 16)

# ClickHouse
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_$(openssl rand -hex 16)
CLICKHOUSE_DB=lurus_logs

# Grafana
GF_SECURITY_ADMIN_PASSWORD=admin_$(openssl rand -hex 8)

# NEW-API
SESSION_SECRET=$(openssl rand -hex 32)
EOF

remote_copy "/tmp/lurus.env" "${REMOTE_DIR}/.env" "Copy environment file"

# Step 8: Pull Docker images
log_info "Step 8: Pulling Docker images..."
remote_exec "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml pull" "Pull images"

# Step 9: Deploy infrastructure services
log_info "Step 9: Deploying infrastructure services (PostgreSQL, Redis, NATS, ClickHouse)..."
remote_exec "cd ${REMOTE_DIR} && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul" "Start infrastructure"

log_warn "Waiting 20 seconds for infrastructure to initialize..."
sleep 20

# Step 10: Check infrastructure health
log_info "Step 10: Checking infrastructure health..."
remote_exec "docker ps --filter 'name=lurus-postgres' --filter 'name=lurus-redis' --filter 'name=lurus-nats' --format 'table {{.Names}}\t{{.Status}}'" "Check infrastructure containers"

# Step 11: Initialize databases
log_info "Step 11: Initializing databases..."
remote_exec "docker exec lurus-postgres psql -U lurus -c 'CREATE DATABASE IF NOT EXISTS new_api;' 2>&1 || echo 'Database may already exist'" "Create NEW-API database"
remote_exec "docker exec lurus-postgres psql -U lurus -c 'CREATE DATABASE IF NOT EXISTS provider;' 2>&1 || echo 'Database may already exist'" "Create Provider database"
remote_exec "docker exec lurus-postgres psql -U lurus -c 'CREATE DATABASE IF NOT EXISTS billing;' 2>&1 || echo 'Database may already exist'" "Create Billing database"
remote_exec "docker exec lurus-postgres psql -U lurus -c 'CREATE DATABASE IF NOT EXISTS sync;' 2>&1 || echo 'Database may already exist'" "Create Sync database"

# Step 12: Display status
log_info "Step 12: Deployment status..."
remote_exec "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'" "Container status"

echo ""
log_info "=== Infrastructure Deployment Complete ==="
echo ""
echo "Next steps:"
echo "1. Build microservices:"
echo "   ./scripts/build-services.sh"
echo ""
echo "2. Deploy microservices:"
echo "   ssh ${SERVER_USER}@${SERVER_IP}"
echo "   cd ${REMOTE_DIR}"
echo "   docker compose -f docker-compose.dev.yaml up -d"
echo ""
echo "3. Check logs:"
echo "   docker logs -f lurus-gateway"
echo ""
echo "4. Access services:"
echo "   - Grafana: http://${SERVER_IP}:3000 (admin/admin)"
echo "   - Prometheus: http://${SERVER_IP}:9090"
echo "   - Jaeger: http://${SERVER_IP}:16686"
echo "   - Consul: http://${SERVER_IP}:8500"
echo ""
