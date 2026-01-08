# Lurus Switch Deployment Guide
# 部署指南

## Quick Start / 快速开始

### Prerequisites / 前置条件

- Docker 24+ with Docker Compose v2
- 8GB RAM minimum (16GB recommended)
- 20GB disk space

### Development Environment / 开发环境

```bash
# Start all services
docker-compose -f docker-compose.dev.yaml up -d

# Check service status
docker-compose -f docker-compose.dev.yaml ps

# View logs
docker-compose -f docker-compose.dev.yaml logs -f gateway-service

# Stop all services
docker-compose -f docker-compose.dev.yaml down
```

---

## Service Ports / 服务端口

| Service | Port | Description |
|---------|------|-------------|
| Gateway | 18100 | HTTP API (Claude/Codex/Gemini) |
| Provider | 18101 | Provider config API |
| Log | 18102 | Log query API |
| Billing | 18103 | Billing/quota API |
| PostgreSQL | 5432 | Main database |
| Redis | 6379 | Cache |
| NATS | 4222/8222 | Message bus |
| ClickHouse | 8123/9000 | Log analytics |
| Consul | 8500 | Service discovery |
| Prometheus | 9090 | Metrics |
| Grafana | 3000 | Dashboards |
| Jaeger | 16686 | Tracing UI |
| Alertmanager | 9093 | Alerts |

---

## Configuration / 配置

### Environment Variables / 环境变量

#### Gateway Service
```bash
PROVIDER_SERVICE_URL=http://provider-service:18101
BILLING_SERVICE_URL=http://billing-service:18103
NATS_URL=nats://nats:4222
TRACING_ENABLED=true
TRACING_ENDPOINT=jaeger:4317
```

#### Provider Service
```bash
DATABASE_DSN=postgres://user:pass@postgres:5432/lurus?sslmode=disable
REDIS_ADDR=redis:6379
NATS_URL=nats://nats:4222
```

#### Billing Service
```bash
DATABASE_DSN=postgres://user:pass@postgres:5432/lurus?sslmode=disable
REDIS_ADDR=redis:6379
NATS_URL=nats://nats:4222
```

#### Log Service
```bash
CLICKHOUSE_DSN=clickhouse://user:pass@clickhouse:9000/lurus_logs
NATS_URL=nats://nats:4222
```

---

## Observability / 可观测性

### Grafana Dashboards / 仪表盘

Access: http://localhost:3000 (admin/admin)

Pre-configured dashboards:
- **Lurus Gateway - LLM Metrics**: Request rates, latency, costs, tokens
- Provider health and failover tracking
- Cache hit rates

### Prometheus Alerts / 告警规则

Access: http://localhost:9090/alerts

Alert groups:
- **service_health**: Service down, high error rate, high latency
- **llm_provider**: Provider errors, failovers
- **cost_usage**: Hourly/daily cost limits, cost spikes
- **request_rate**: Traffic anomalies
- **cache**: Low hit rate

### Jaeger Tracing / 链路追踪

Access: http://localhost:16686

Services traced:
- gateway-service
- provider-service
- billing-service
- log-service

---

## Health Checks / 健康检查

```bash
# Gateway
curl http://localhost:18100/health

# Provider
curl http://localhost:18101/health

# Billing
curl http://localhost:18103/health

# Log
curl http://localhost:18102/health
```

---

## Database Initialization / 数据库初始化

### PostgreSQL
```bash
# Schema is auto-created via init-db.sql
docker exec -it lurus-postgres psql -U lurus -d lurus -c "\dt"
```

### ClickHouse
```bash
# Schema is auto-created via clickhouse-init.sql
docker exec -it lurus-clickhouse clickhouse-client --query "SHOW TABLES FROM lurus_logs"
```

---

## Scaling / 扩展

### Horizontal Scaling
```bash
# Scale gateway instances
docker-compose -f docker-compose.dev.yaml up -d --scale gateway-service=3
```

### Resource Limits (Production)
```yaml
services:
  gateway-service:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

---

## Troubleshooting / 故障排查

### Common Issues / 常见问题

1. **Service won't start**
   ```bash
   # Check logs
   docker-compose -f docker-compose.dev.yaml logs <service-name>

   # Check dependencies
   docker-compose -f docker-compose.dev.yaml ps
   ```

2. **Database connection failed**
   ```bash
   # Verify PostgreSQL is healthy
   docker exec lurus-postgres pg_isready -U lurus
   ```

3. **NATS connection issues**
   ```bash
   # Check NATS server
   curl http://localhost:8222/varz
   ```

4. **High latency**
   - Check Jaeger for slow spans
   - Check Prometheus for resource metrics
   - Review provider failover rates

### Logs Location / 日志位置

```bash
# Container logs
docker logs lurus-gateway

# ClickHouse (historical)
docker exec lurus-clickhouse clickhouse-client --query "SELECT * FROM lurus_logs.request_log LIMIT 10"
```

---

## Backup & Recovery / 备份与恢复

### PostgreSQL Backup
```bash
# Backup
docker exec lurus-postgres pg_dump -U lurus lurus > backup.sql

# Restore
docker exec -i lurus-postgres psql -U lurus lurus < backup.sql
```

### ClickHouse Backup
```bash
# Backup
docker exec lurus-clickhouse clickhouse-client --query "BACKUP TABLE lurus_logs.request_log TO Disk('backups', 'request_log.zip')"
```

---

## Security Notes / 安全注意事项

1. **Change default passwords** before production deployment
2. **Enable TLS** for all external endpoints
3. **Configure firewall** to restrict database access
4. **Rotate API keys** regularly
5. **Enable audit logging** in production

---

## Architecture Overview / 架构概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Lurus Switch Microservices                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Gateway   │  │  Provider   │  │     Log     │  │   Billing   │        │
│  │   :18100    │  │   :18101    │  │   :18102    │  │   :18103    │        │
│  │   Hertz     │  │   Kratos    │  │   Kratos    │  │   Kratos    │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │               │
│         └────────────────┴────────────────┴────────────────┘               │
│                                   │                                         │
│                          NATS JetStream                                     │
│                                   │                                         │
│         ┌─────────────────────────┼─────────────────────────┐               │
│         ▼                         ▼                         ▼               │
│   PostgreSQL              ClickHouse                    Redis               │
│   (Config/Billing)        (Logs/Analytics)             (Cache)              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

*Generated: 2026-01-08*
*Version: 1.0.0*
