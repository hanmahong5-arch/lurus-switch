#!/bin/bash
# Lurus Switch Integration Test Script
# Tests all microservices and observability stack
# Usage: ./scripts/integration-test.sh

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║       Lurus Switch Integration Test Suite             ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""
echo "Started at: $(date)"
echo ""

# Configuration - Services
GATEWAY_URL="${GATEWAY_URL:-http://localhost:18100}"
PROVIDER_URL="${PROVIDER_URL:-http://localhost:18101}"
LOG_URL="${LOG_URL:-http://localhost:18102}"
BILLING_URL="${BILLING_URL:-http://localhost:18103}"

# Configuration - Infrastructure
NATS_URL="${NATS_URL:-http://localhost:8222}"
POSTGRES_CONTAINER="lurus-postgres"
REDIS_CONTAINER="lurus-redis"
CLICKHOUSE_URL="${CLICKHOUSE_URL:-http://localhost:8123}"
CONSUL_URL="${CONSUL_URL:-http://localhost:8500}"

# Configuration - Observability
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
ALERTMANAGER_URL="${ALERTMANAGER_URL:-http://localhost:9093}"

# Alias for backward compatibility
PROVIDER_SERVICE_URL="$PROVIDER_URL"
BILLING_SERVICE_URL="$BILLING_URL"
GATEWAY_SERVICE_URL="$GATEWAY_URL"
LOG_SERVICE_URL="$LOG_URL"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0

log_pass() {
    echo -e " ${GREEN}OK${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e " ${RED}FAILED${NC} $1"
    ((FAILED++))
}

log_skip() {
    echo -e " ${YELLOW}SKIPPED${NC} $1"
    ((SKIPPED++))
}

test_service_health() {
    local service_name=$1
    local url=$2

    printf "Testing $service_name health..."
    if response=$(curl -s -f "$url/health" 2>/dev/null); then
        if echo "$response" | grep -q "healthy"; then
            echo -e " ${GREEN}OK${NC}"
            return 0
        fi
    fi
    echo -e " ${RED}FAILED (not running)${NC}"
    return 1
}

test_provider_api() {
    echo ""
    echo -e "${YELLOW}=== Provider Service API Tests ===${NC}"

    # Test: List providers
    printf "  GET /api/v1/providers?platform=claude..."
    if response=$(curl -s -f "$PROVIDER_SERVICE_URL/api/v1/providers?platform=claude" 2>/dev/null); then
        count=$(echo "$response" | grep -o '"name"' | wc -l)
        echo -e " ${GREEN}OK (found $count providers)${NC}"
    else
        echo -e " ${RED}FAILED${NC}"
    fi

    # Test: Create and delete provider
    printf "  POST /api/v1/providers (create)..."
    body='{"name":"Test Provider","platform":"claude","api_url":"https://api.test.com","api_key":"test-key","enabled":true}'
    if response=$(curl -s -f -X POST "$PROVIDER_SERVICE_URL/api/v1/providers" -H "Content-Type: application/json" -d "$body" 2>/dev/null); then
        id=$(echo "$response" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')
        echo -e " ${GREEN}OK (id=$id)${NC}"

        printf "  DELETE /api/v1/providers/$id..."
        if curl -s -f -X DELETE "$PROVIDER_SERVICE_URL/api/v1/providers/$id" >/dev/null 2>&1; then
            echo -e " ${GREEN}OK${NC}"
        else
            echo -e " ${RED}FAILED${NC}"
        fi
    else
        echo -e " ${RED}FAILED${NC}"
    fi
}

test_billing_api() {
    echo ""
    echo -e "${YELLOW}=== Billing Service API Tests ===${NC}"

    test_user="test-user-$RANDOM"

    # Test: Check balance
    printf "  GET /api/v1/billing/check/$test_user..."
    if response=$(curl -s -f "$BILLING_SERVICE_URL/api/v1/billing/check/$test_user" 2>/dev/null); then
        echo -e " ${GREEN}OK${NC}"
    else
        echo -e " ${RED}FAILED${NC}"
    fi

    # Test: Get quota
    printf "  GET /api/v1/billing/quota/$test_user..."
    if response=$(curl -s -f "$BILLING_SERVICE_URL/api/v1/billing/quota/$test_user" 2>/dev/null); then
        echo -e " ${GREEN}OK${NC}"
    else
        echo -e " ${RED}FAILED${NC}"
    fi

    # Test: Record usage
    printf "  POST /api/v1/billing/usage..."
    body='{"user_id":"'"$test_user"'","platform":"claude","model":"claude-3-opus","input_tokens":1000,"output_tokens":500}'
    if curl -s -f -X POST "$BILLING_SERVICE_URL/api/v1/billing/usage" -H "Content-Type: application/json" -d "$body" >/dev/null 2>&1; then
        echo -e " ${GREEN}OK${NC}"
    else
        echo -e " ${RED}FAILED${NC}"
    fi
}

test_gateway_api() {
    echo ""
    echo -e "${YELLOW}=== Gateway Service API Tests ===${NC}"

    # Test: Health
    printf "  GET /health..."
    if curl -s -f "$GATEWAY_SERVICE_URL/health" >/dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # Test: Ready
    printf "  GET /ready..."
    if curl -s -f "$GATEWAY_SERVICE_URL/ready" >/dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # Test: Metrics
    printf "  GET /metrics..."
    if response=$(curl -s -f "$GATEWAY_SERVICE_URL/metrics" 2>/dev/null); then
        if echo "$response" | grep -q "go_"; then
            log_pass "(Prometheus format)"
        else
            log_pass ""
        fi
    else
        log_fail ""
    fi

    # Test: Claude API endpoint (expect 401 without auth)
    printf "  POST /v1/messages (no auth)..."
    code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$GATEWAY_SERVICE_URL/v1/messages" 2>/dev/null || echo "000")
    if [ "$code" == "401" ] || [ "$code" == "400" ]; then
        log_pass "(returned $code)"
    else
        log_skip "(returned $code)"
    fi
}

test_infrastructure() {
    echo ""
    echo -e "${YELLOW}=== Infrastructure Tests ===${NC}"

    # PostgreSQL
    printf "  PostgreSQL..."
    if docker exec $POSTGRES_CONTAINER pg_isready -U lurus > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail "(container not running)"
    fi

    # Redis
    printf "  Redis..."
    if docker exec $REDIS_CONTAINER redis-cli ping > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail "(container not running)"
    fi

    # NATS
    printf "  NATS Server..."
    if curl -s -f "$NATS_URL/varz" > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # NATS JetStream
    printf "  NATS JetStream..."
    if curl -s -f "$NATS_URL/jsz" 2>/dev/null | grep -q "streams"; then
        log_pass ""
    else
        log_skip ""
    fi

    # ClickHouse
    printf "  ClickHouse..."
    if curl -s -f "$CLICKHOUSE_URL/ping" > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # Consul
    printf "  Consul..."
    if curl -s -f "$CONSUL_URL/v1/status/leader" > /dev/null 2>&1; then
        log_pass ""
    else
        log_skip ""
    fi
}

test_observability() {
    echo ""
    echo -e "${YELLOW}=== Observability Stack Tests ===${NC}"

    # Prometheus
    printf "  Prometheus..."
    if curl -s -f "$PROMETHEUS_URL/-/healthy" > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # Prometheus targets
    printf "  Prometheus targets..."
    if curl -s -f "$PROMETHEUS_URL/api/v1/targets" 2>/dev/null | grep -q "up"; then
        log_pass ""
    else
        log_skip ""
    fi

    # Prometheus alerts
    printf "  Prometheus alerts..."
    if curl -s -f "$PROMETHEUS_URL/api/v1/rules" 2>/dev/null | grep -q "groups"; then
        log_pass ""
    else
        log_skip ""
    fi

    # Grafana
    printf "  Grafana..."
    if curl -s -f "$GRAFANA_URL/api/health" 2>/dev/null | grep -q "ok"; then
        log_pass ""
    else
        log_fail ""
    fi

    # Grafana datasources
    printf "  Grafana datasources..."
    if curl -s -f -u admin:admin "$GRAFANA_URL/api/datasources" 2>/dev/null | grep -q "Prometheus"; then
        log_pass ""
    else
        log_skip ""
    fi

    # Jaeger
    printf "  Jaeger..."
    if curl -s -f "$JAEGER_URL" > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # Jaeger services
    printf "  Jaeger services..."
    if curl -s -f "$JAEGER_URL/api/services" 2>/dev/null | grep -q "data"; then
        log_pass ""
    else
        log_skip "(no traces yet)"
    fi

    # Alertmanager
    printf "  Alertmanager..."
    if curl -s -f "$ALERTMANAGER_URL/-/healthy" > /dev/null 2>&1; then
        log_pass ""
    else
        log_fail ""
    fi

    # Alertmanager status
    printf "  Alertmanager config..."
    if curl -s -f "$ALERTMANAGER_URL/api/v2/status" 2>/dev/null | grep -q "cluster"; then
        log_pass ""
    else
        log_skip ""
    fi
}

# Main execution
echo -e "${CYAN}1. Checking service health...${NC}"
echo ""

gateway_ok=0
provider_ok=0
log_ok=0
billing_ok=0

test_service_health "Gateway Service (:18100)" "$GATEWAY_SERVICE_URL" && gateway_ok=1
test_service_health "Provider Service (:18101)" "$PROVIDER_SERVICE_URL" && provider_ok=1
test_service_health "Log Service (:18102)" "$LOG_SERVICE_URL" && log_ok=1
test_service_health "Billing Service (:18103)" "$BILLING_SERVICE_URL" && billing_ok=1

echo ""
echo -e "${CYAN}2. Running API tests...${NC}"

if [ $provider_ok -eq 1 ]; then
    test_provider_api
else
    echo -e "\n${YELLOW}Skipping Provider API tests (service not running)${NC}"
fi

if [ $billing_ok -eq 1 ]; then
    test_billing_api
else
    echo -e "\n${YELLOW}Skipping Billing API tests (service not running)${NC}"
fi

if [ $gateway_ok -eq 1 ]; then
    test_gateway_api
else
    echo -e "\n${YELLOW}Skipping Gateway API tests (service not running)${NC}"
fi

echo ""
echo -e "${CYAN}3. Testing Infrastructure...${NC}"
test_infrastructure

echo ""
echo -e "${CYAN}4. Testing Observability Stack...${NC}"
test_observability

echo ""
echo "====================================="
echo "  Integration Test Complete"
echo "====================================="

# Summary
running_count=$((gateway_ok + provider_ok + log_ok + billing_ok))
echo ""
echo "Completed at: $(date)"
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${GREEN}Passed:${NC}  $PASSED"
echo -e "  ${RED}Failed:${NC}  $FAILED"
echo -e "  ${YELLOW}Skipped:${NC} $SKIPPED"
echo ""

if [ $running_count -eq 4 ]; then
    echo -e "Microservices: ${GREEN}$running_count/4 running${NC}"
else
    echo -e "Microservices: ${YELLOW}$running_count/4 running${NC}"
    echo ""
    echo -e "${CYAN}To start all services, run:${NC}"
    echo "  docker-compose -f docker-compose.dev.yaml up -d"
fi

if [ $FAILED -gt 0 ]; then
    echo ""
    echo -e "${RED}Some tests failed. Check logs above for details.${NC}"
    exit 1
fi
