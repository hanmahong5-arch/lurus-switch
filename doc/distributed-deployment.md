# Lurus Switch 分布式部署指南 / Distributed Deployment Guide

> 版本: v1.0 | 创建日期: 2026-01-08

本文档描述如何将 Lurus Switch 部署为高可用分布式架构。

---

## 目录

1. [架构概览](#架构概览)
2. [硬件规划](#硬件规划)
3. [网络拓扑](#网络拓扑)
4. [PostgreSQL 主备](#postgresql-主备)
5. [Redis 集群](#redis-集群)
6. [NATS 集群](#nats-集群)
7. [应用服务集群](#应用服务集群)
8. [负载均衡](#负载均衡)
9. [服务发现](#服务发现)
10. [分布式监控](#分布式监控)
11. [Docker Compose 配置](#docker-compose-配置)
12. [K3S 高可用配置](#k3s-高可用配置)

---

## 架构概览

### 目标架构

```
                                    ┌─────────────────────────────────────┐
                                    │           Load Balancer             │
                                    │   (Nginx/HAProxy/Cloud LB)          │
                                    │         VIP: 10.0.0.100             │
                                    └───────────────┬─────────────────────┘
                                                    │
                    ┌───────────────────────────────┼───────────────────────────────┐
                    │                               │                               │
                    ▼                               ▼                               ▼
           ┌─────────────────┐             ┌─────────────────┐             ┌─────────────────┐
           │   Node 1        │             │   Node 2        │             │   Node 3        │
           │   10.0.0.1      │             │   10.0.0.2      │             │   10.0.0.3      │
           │                 │             │                 │             │                 │
           │ ┌─────────────┐ │             │ ┌─────────────┐ │             │ ┌─────────────┐ │
           │ │ Caddy       │ │             │ │ Caddy       │ │             │ │ Caddy       │ │
           │ └─────────────┘ │             │ └─────────────┘ │             │ └─────────────┘ │
           │ ┌─────────────┐ │             │ ┌─────────────┐ │             │ ┌─────────────┐ │
           │ │ Gateway ×2  │ │             │ │ Gateway ×2  │ │             │ │ Gateway ×2  │ │
           │ └─────────────┘ │             │ └─────────────┘ │             │ └─────────────┘ │
           │ ┌─────────────┐ │             │ ┌─────────────┐ │             │ ┌─────────────┐ │
           │ │ Provider    │ │             │ │ Billing     │ │             │ │ Log         │ │
           │ │ NEW-API     │ │             │ │ Sync        │ │             │ │ NEW-API     │ │
           │ └─────────────┘ │             │ └─────────────┘ │             │ └─────────────┘ │
           └────────┬────────┘             └────────┬────────┘             └────────┬────────┘
                    │                               │                               │
                    └───────────────────────────────┼───────────────────────────────┘
                                                    │
        ┌───────────────────────────────────────────┼───────────────────────────────────────────┐
        │                                           │                                           │
        │                              Internal Network (10.0.1.0/24)                           │
        │                                           │                                           │
        │   ┌───────────────────┐    ┌──────────────┴──────────────┐    ┌───────────────────┐   │
        │   │                   │    │                             │    │                   │   │
        │   │  PostgreSQL       │    │       NATS Cluster          │    │   Redis Cluster   │   │
        │   │  Primary/Standby  │    │       (3 nodes)             │    │   (6 nodes)       │   │
        │   │                   │    │                             │    │                   │   │
        │   │ ┌───────────────┐ │    │ ┌─────┐ ┌─────┐ ┌─────┐    │    │ ┌─────┐ ┌─────┐   │   │
        │   │ │ Primary       │ │    │ │NATS1│ │NATS2│ │NATS3│    │    │ │ M1  │ │ M2  │   │   │
        │   │ │ 10.0.1.10     │ │    │ │:4222│ │:4222│ │:4222│    │    │ │:7001│ │:7002│   │   │
        │   │ └───────┬───────┘ │    │ └─────┘ └─────┘ └─────┘    │    │ └──┬──┘ └──┬──┘   │   │
        │   │         │ Repl    │    │     10.0.1.21-23           │    │    │       │      │   │
        │   │ ┌───────▼───────┐ │    └─────────────────────────────┘    │ ┌──▼──┐ ┌──▼──┐   │   │
        │   │ │ Standby       │ │                                       │ │ S1  │ │ S2  │   │   │
        │   │ │ 10.0.1.11     │ │                                       │ │:7003│ │:7004│   │   │
        │   │ └───────────────┘ │                                       │ └─────┘ └─────┘   │   │
        │   │                   │                                       │ ┌─────┐ ┌─────┐   │   │
        │   │ PgBouncer Pool    │                                       │ │ M3  │ │ S3  │   │   │
        │   │ 10.0.1.12:6432    │                                       │ │:7005│ │:7006│   │   │
        │   └───────────────────┘                                       │ └─────┘ └─────┘   │   │
        │                                                               │  10.0.1.31-36    │   │
        │                                                               └───────────────────┘   │
        └───────────────────────────────────────────────────────────────────────────────────────┘
```

### 组件高可用级别

| 组件 | 最小节点数 | 推荐节点数 | 故障容忍 |
|------|-----------|-----------|---------|
| PostgreSQL | 2 (主备) | 3 (1主2备) | 1 节点 |
| Redis Cluster | 6 (3主3从) | 6 | 1 主节点 |
| NATS Cluster | 3 | 3-5 | 1 节点 |
| Gateway Service | 2 | 3+ | N-1 节点 |
| 其他微服务 | 2 | 2+ | N-1 节点 |

---

## 硬件规划

### 最小生产配置 (3 节点)

| 节点 | CPU | 内存 | 存储 | 角色 |
|------|-----|------|------|------|
| node-1 | 4 核 | 16 GB | 100 GB SSD | App + PG Primary + Redis M1/S2 + NATS1 |
| node-2 | 4 核 | 16 GB | 100 GB SSD | App + PG Standby + Redis M2/S3 + NATS2 |
| node-3 | 4 核 | 16 GB | 100 GB SSD | App + PgBouncer + Redis M3/S1 + NATS3 |

### 推荐生产配置 (分离式)

| 节点组 | 数量 | CPU | 内存 | 存储 | 角色 |
|--------|------|-----|------|------|------|
| app-* | 3+ | 4 核 | 8 GB | 50 GB | 应用服务 |
| pg-* | 2-3 | 4 核 | 16 GB | 200 GB NVMe | PostgreSQL |
| redis-* | 6 | 2 核 | 8 GB | 50 GB | Redis Cluster |
| nats-* | 3 | 2 核 | 4 GB | 20 GB | NATS Cluster |
| mon-* | 1-2 | 4 核 | 16 GB | 500 GB | 监控 (Prometheus/Grafana/ClickHouse) |

### 云服务商推荐规格

**阿里云**:
- 应用节点: ecs.c7.xlarge (4C8G)
- 数据库节点: ecs.r7.xlarge (4C16G)
- Redis: ecs.r7.large (2C8G)

**AWS**:
- 应用节点: c6i.xlarge
- 数据库节点: r6i.xlarge
- Redis: r6i.large

---

## 网络拓扑

### IP 规划示例

```
# 公网 (仅 LB)
LB VIP: 47.x.x.x (公网 IP)

# 应用网络 10.0.0.0/24
app-1: 10.0.0.1
app-2: 10.0.0.2
app-3: 10.0.0.3

# 数据网络 10.0.1.0/24 (内网隔离)
pg-primary:   10.0.1.10
pg-standby:   10.0.1.11
pgbouncer:    10.0.1.12

nats-1:       10.0.1.21
nats-2:       10.0.1.22
nats-3:       10.0.1.23

redis-1:      10.0.1.31
redis-2:      10.0.1.32
redis-3:      10.0.1.33
redis-4:      10.0.1.34
redis-5:      10.0.1.35
redis-6:      10.0.1.36

# 监控网络 10.0.2.0/24
prometheus:   10.0.2.1
grafana:      10.0.2.2
clickhouse:   10.0.2.3
```

### 防火墙规则

```bash
# 应用节点开放
- 80/443 (HTTP/HTTPS) - 仅从 LB
- 18100-18103 (微服务) - 内网

# PostgreSQL
- 5432 - 仅内网
- 6432 (PgBouncer) - 仅内网

# Redis Cluster
- 7001-7006 - 仅内网
- 17001-17006 (Bus) - 仅内网

# NATS
- 4222 (Client) - 仅内网
- 6222 (Cluster) - 仅内网
- 8222 (Monitor) - 仅内网
```

---

## PostgreSQL 主备

### 方案选择

| 方案 | 复杂度 | 自动故障转移 | 推荐场景 |
|------|--------|-------------|---------|
| 流复制 + 手动切换 | 低 | ❌ | 开发/测试 |
| Patroni + etcd | 中 | ✅ | 生产推荐 |
| Citus 分布式 | 高 | ✅ | 超大规模 |
| 云托管 RDS | 低 | ✅ | 省心首选 |

### Patroni 高可用方案

```yaml
# docker-compose.pg-ha.yml
version: '3.8'

services:
  etcd1:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: etcd1
    command:
      - etcd
      - --name=etcd1
      - --initial-advertise-peer-urls=http://etcd1:2380
      - --listen-peer-urls=http://0.0.0.0:2380
      - --listen-client-urls=http://0.0.0.0:2379
      - --advertise-client-urls=http://etcd1:2379
      - --initial-cluster=etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380
      - --initial-cluster-state=new
    networks:
      - pg-cluster

  etcd2:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: etcd2
    command:
      - etcd
      - --name=etcd2
      - --initial-advertise-peer-urls=http://etcd2:2380
      - --listen-peer-urls=http://0.0.0.0:2380
      - --listen-client-urls=http://0.0.0.0:2379
      - --advertise-client-urls=http://etcd2:2379
      - --initial-cluster=etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380
      - --initial-cluster-state=new
    networks:
      - pg-cluster

  etcd3:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: etcd3
    command:
      - etcd
      - --name=etcd3
      - --initial-advertise-peer-urls=http://etcd3:2380
      - --listen-peer-urls=http://0.0.0.0:2380
      - --listen-client-urls=http://0.0.0.0:2379
      - --advertise-client-urls=http://etcd3:2379
      - --initial-cluster=etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380
      - --initial-cluster-state=new
    networks:
      - pg-cluster

  patroni1:
    image: patroni/patroni:3.2.0
    container_name: patroni1
    hostname: patroni1
    environment:
      PATRONI_NAME: patroni1
      PATRONI_SCOPE: lurus-pg
      PATRONI_RESTAPI_CONNECT_ADDRESS: patroni1:8008
      PATRONI_RESTAPI_LISTEN: 0.0.0.0:8008
      PATRONI_ETCD3_HOSTS: etcd1:2379,etcd2:2379,etcd3:2379
      PATRONI_POSTGRESQL_CONNECT_ADDRESS: patroni1:5432
      PATRONI_POSTGRESQL_LISTEN: 0.0.0.0:5432
      PATRONI_POSTGRESQL_DATA_DIR: /var/lib/postgresql/data
      PATRONI_SUPERUSER_USERNAME: postgres
      PATRONI_SUPERUSER_PASSWORD: postgres
      PATRONI_REPLICATION_USERNAME: replicator
      PATRONI_REPLICATION_PASSWORD: replicator
    volumes:
      - patroni1-data:/var/lib/postgresql/data
    networks:
      - pg-cluster
    depends_on:
      - etcd1
      - etcd2
      - etcd3

  patroni2:
    image: patroni/patroni:3.2.0
    container_name: patroni2
    hostname: patroni2
    environment:
      PATRONI_NAME: patroni2
      PATRONI_SCOPE: lurus-pg
      PATRONI_RESTAPI_CONNECT_ADDRESS: patroni2:8008
      PATRONI_RESTAPI_LISTEN: 0.0.0.0:8008
      PATRONI_ETCD3_HOSTS: etcd1:2379,etcd2:2379,etcd3:2379
      PATRONI_POSTGRESQL_CONNECT_ADDRESS: patroni2:5432
      PATRONI_POSTGRESQL_LISTEN: 0.0.0.0:5432
      PATRONI_POSTGRESQL_DATA_DIR: /var/lib/postgresql/data
      PATRONI_SUPERUSER_USERNAME: postgres
      PATRONI_SUPERUSER_PASSWORD: postgres
      PATRONI_REPLICATION_USERNAME: replicator
      PATRONI_REPLICATION_PASSWORD: replicator
    volumes:
      - patroni2-data:/var/lib/postgresql/data
    networks:
      - pg-cluster
    depends_on:
      - etcd1
      - etcd2
      - etcd3

  patroni3:
    image: patroni/patroni:3.2.0
    container_name: patroni3
    hostname: patroni3
    environment:
      PATRONI_NAME: patroni3
      PATRONI_SCOPE: lurus-pg
      PATRONI_RESTAPI_CONNECT_ADDRESS: patroni3:8008
      PATRONI_RESTAPI_LISTEN: 0.0.0.0:8008
      PATRONI_ETCD3_HOSTS: etcd1:2379,etcd2:2379,etcd3:2379
      PATRONI_POSTGRESQL_CONNECT_ADDRESS: patroni3:5432
      PATRONI_POSTGRESQL_LISTEN: 0.0.0.0:5432
      PATRONI_POSTGRESQL_DATA_DIR: /var/lib/postgresql/data
      PATRONI_SUPERUSER_USERNAME: postgres
      PATRONI_SUPERUSER_PASSWORD: postgres
      PATRONI_REPLICATION_USERNAME: replicator
      PATRONI_REPLICATION_PASSWORD: replicator
    volumes:
      - patroni3-data:/var/lib/postgresql/data
    networks:
      - pg-cluster
    depends_on:
      - etcd1
      - etcd2
      - etcd3

  # HAProxy for PostgreSQL load balancing
  haproxy:
    image: haproxy:2.9-alpine
    container_name: pg-haproxy
    ports:
      - "5432:5432"   # Primary (read-write)
      - "5433:5433"   # Replicas (read-only)
      - "8404:8404"   # Stats
    volumes:
      - ./haproxy/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
    networks:
      - pg-cluster
    depends_on:
      - patroni1
      - patroni2
      - patroni3

volumes:
  patroni1-data:
  patroni2-data:
  patroni3-data:

networks:
  pg-cluster:
    driver: bridge
```

### HAProxy 配置 (PostgreSQL)

```haproxy
# haproxy/haproxy.cfg
global
    maxconn 1000

defaults
    mode tcp
    timeout connect 10s
    timeout client 30s
    timeout server 30s

listen stats
    bind *:8404
    mode http
    stats enable
    stats uri /stats
    stats refresh 10s

# Primary (read-write)
listen postgres-primary
    bind *:5432
    option httpchk GET /primary
    http-check expect status 200
    default-server inter 3s fall 3 rise 2 on-marked-down shutdown-sessions
    server patroni1 patroni1:5432 check port 8008
    server patroni2 patroni2:5432 check port 8008
    server patroni3 patroni3:5432 check port 8008

# Replicas (read-only, load balanced)
listen postgres-replicas
    bind *:5433
    balance roundrobin
    option httpchk GET /replica
    http-check expect status 200
    default-server inter 3s fall 3 rise 2 on-marked-down shutdown-sessions
    server patroni1 patroni1:5432 check port 8008
    server patroni2 patroni2:5432 check port 8008
    server patroni3 patroni3:5432 check port 8008
```

### 应用连接配置

```yaml
# 应用配置
database:
  # 写操作连接主节点
  primary_dsn: "postgresql://lurus:password@haproxy:5432/lurus_billing?sslmode=disable"
  # 读操作连接从节点 (负载均衡)
  replica_dsn: "postgresql://lurus:password@haproxy:5433/lurus_billing?sslmode=disable"
```

---

## Redis 集群

### Redis Cluster 模式 (推荐)

```yaml
# docker-compose.redis-cluster.yml
version: '3.8'

services:
  redis-1:
    image: redis:7-alpine
    container_name: redis-1
    command: redis-server /etc/redis/redis.conf
    ports:
      - "7001:7001"
      - "17001:17001"
    volumes:
      - ./redis/redis-1.conf:/etc/redis/redis.conf
      - redis-1-data:/data
    networks:
      - redis-cluster

  redis-2:
    image: redis:7-alpine
    container_name: redis-2
    command: redis-server /etc/redis/redis.conf
    ports:
      - "7002:7002"
      - "17002:17002"
    volumes:
      - ./redis/redis-2.conf:/etc/redis/redis.conf
      - redis-2-data:/data
    networks:
      - redis-cluster

  redis-3:
    image: redis:7-alpine
    container_name: redis-3
    command: redis-server /etc/redis/redis.conf
    ports:
      - "7003:7003"
      - "17003:17003"
    volumes:
      - ./redis/redis-3.conf:/etc/redis/redis.conf
      - redis-3-data:/data
    networks:
      - redis-cluster

  redis-4:
    image: redis:7-alpine
    container_name: redis-4
    command: redis-server /etc/redis/redis.conf
    ports:
      - "7004:7004"
      - "17004:17004"
    volumes:
      - ./redis/redis-4.conf:/etc/redis/redis.conf
      - redis-4-data:/data
    networks:
      - redis-cluster

  redis-5:
    image: redis:7-alpine
    container_name: redis-5
    command: redis-server /etc/redis/redis.conf
    ports:
      - "7005:7005"
      - "17005:17005"
    volumes:
      - ./redis/redis-5.conf:/etc/redis/redis.conf
      - redis-5-data:/data
    networks:
      - redis-cluster

  redis-6:
    image: redis:7-alpine
    container_name: redis-6
    command: redis-server /etc/redis/redis.conf
    ports:
      - "7006:7006"
      - "17006:17006"
    volumes:
      - ./redis/redis-6.conf:/etc/redis/redis.conf
      - redis-6-data:/data
    networks:
      - redis-cluster

volumes:
  redis-1-data:
  redis-2-data:
  redis-3-data:
  redis-4-data:
  redis-5-data:
  redis-6-data:

networks:
  redis-cluster:
    driver: bridge
```

### Redis 节点配置

```conf
# redis/redis-1.conf (每个节点类似，改端口)
port 7001
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000
appendonly yes
requirepass lurus_dev_2024
masterauth lurus_dev_2024
```

### 初始化集群

```bash
# 启动所有节点后执行
docker exec -it redis-1 redis-cli -a lurus_dev_2024 --cluster create \
  redis-1:7001 redis-2:7002 redis-3:7003 \
  redis-4:7004 redis-5:7005 redis-6:7006 \
  --cluster-replicas 1 --cluster-yes

# 验证集群
docker exec -it redis-1 redis-cli -a lurus_dev_2024 -p 7001 cluster info
docker exec -it redis-1 redis-cli -a lurus_dev_2024 -p 7001 cluster nodes
```

### 应用连接配置

```go
// Go Redis Cluster 连接
import "github.com/redis/go-redis/v9"

rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{
        "redis-1:7001",
        "redis-2:7002",
        "redis-3:7003",
        "redis-4:7004",
        "redis-5:7005",
        "redis-6:7006",
    },
    Password: "lurus_dev_2024",
})
```

---

## NATS 集群

### NATS JetStream 集群

```yaml
# docker-compose.nats-cluster.yml
version: '3.8'

services:
  nats-1:
    image: nats:2.10-alpine
    container_name: nats-1
    command:
      - "--config=/etc/nats/nats.conf"
      - "--name=nats-1"
    ports:
      - "4222:4222"
      - "6222:6222"
      - "8222:8222"
    volumes:
      - ./nats/nats-cluster.conf:/etc/nats/nats.conf
      - nats-1-data:/data
    networks:
      - nats-cluster

  nats-2:
    image: nats:2.10-alpine
    container_name: nats-2
    command:
      - "--config=/etc/nats/nats.conf"
      - "--name=nats-2"
    ports:
      - "4223:4222"
      - "6223:6222"
      - "8223:8222"
    volumes:
      - ./nats/nats-cluster.conf:/etc/nats/nats.conf
      - nats-2-data:/data
    networks:
      - nats-cluster

  nats-3:
    image: nats:2.10-alpine
    container_name: nats-3
    command:
      - "--config=/etc/nats/nats.conf"
      - "--name=nats-3"
    ports:
      - "4224:4222"
      - "6224:6222"
      - "8224:8222"
    volumes:
      - ./nats/nats-cluster.conf:/etc/nats/nats.conf
      - nats-3-data:/data
    networks:
      - nats-cluster

volumes:
  nats-1-data:
  nats-2-data:
  nats-3-data:

networks:
  nats-cluster:
    driver: bridge
```

### NATS 集群配置

```conf
# nats/nats-cluster.conf
server_name: $NATS_SERVER_NAME

listen: 0.0.0.0:4222
http: 0.0.0.0:8222

cluster {
  name: lurus-nats
  listen: 0.0.0.0:6222

  routes: [
    nats-route://nats-1:6222
    nats-route://nats-2:6222
    nats-route://nats-3:6222
  ]
}

jetstream {
  store_dir: /data/jetstream
  max_mem: 1G
  max_file: 10G
}

# 监控
server_tags: ["region:cn-north"]
```

### 应用连接配置

```go
// Go NATS 集群连接
import "github.com/nats-io/nats.go"

nc, err := nats.Connect(
    "nats://nats-1:4222,nats://nats-2:4222,nats://nats-3:4222",
    nats.MaxReconnects(-1),
    nats.ReconnectWait(2*time.Second),
)
```

---

## 应用服务集群

### 无状态服务扩展

```yaml
# docker-compose.app-cluster.yml
version: '3.8'

services:
  gateway-service:
    image: ghcr.io/your-org/lurus-gateway:latest
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
    environment:
      - NATS_URL=nats://nats-1:4222,nats://nats-2:4222,nats://nats-3:4222
      - REDIS_CLUSTER=redis-1:7001,redis-2:7002,redis-3:7003
      - PG_PRIMARY=haproxy:5432
      - PG_REPLICA=haproxy:5433
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:18100/health"]
      interval: 10s
      timeout: 5s
      retries: 3
    networks:
      - app-network

  provider-service:
    image: ghcr.io/your-org/lurus-provider:latest
    deploy:
      replicas: 2
    # ... similar config

  billing-service:
    image: ghcr.io/your-org/lurus-billing:latest
    deploy:
      replicas: 2
    # ... similar config

networks:
  app-network:
    external: true
```

---

## 负载均衡

### Nginx 负载均衡配置

```nginx
# nginx/nginx.conf
upstream gateway_cluster {
    least_conn;
    server app-1:18100 weight=1;
    server app-2:18100 weight=1;
    server app-3:18100 weight=1;
    keepalive 32;
}

upstream newapi_cluster {
    least_conn;
    server app-1:3000 weight=1;
    server app-2:3000 weight=1;
    server app-3:3000 weight=1;
    keepalive 32;
}

server {
    listen 80;
    server_name api.lurus.cn;

    location / {
        proxy_pass http://newapi_cluster;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

server {
    listen 80;
    server_name ai.lurus.cn;

    location /v1/ {
        proxy_pass http://gateway_cluster;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;  # SSE support
    }
}
```

### 健康检查端点

```go
// 服务需要实现详细的健康检查
func healthCheck(c *gin.Context) {
    status := gin.H{
        "status": "healthy",
        "checks": gin.H{
            "database": checkDB(),
            "redis":    checkRedis(),
            "nats":     checkNATS(),
        },
    }
    c.JSON(http.StatusOK, status)
}
```

---

## 服务发现

### Consul 服务发现

```yaml
# docker-compose.consul.yml
version: '3.8'

services:
  consul-server-1:
    image: hashicorp/consul:1.17
    container_name: consul-1
    command: agent -server -bootstrap-expect=3 -ui -client=0.0.0.0 -datacenter=dc1
    environment:
      - CONSUL_BIND_INTERFACE=eth0
    volumes:
      - consul-1-data:/consul/data
    networks:
      - consul-network

  consul-server-2:
    image: hashicorp/consul:1.17
    container_name: consul-2
    command: agent -server -retry-join=consul-1 -client=0.0.0.0 -datacenter=dc1
    environment:
      - CONSUL_BIND_INTERFACE=eth0
    volumes:
      - consul-2-data:/consul/data
    networks:
      - consul-network

  consul-server-3:
    image: hashicorp/consul:1.17
    container_name: consul-3
    command: agent -server -retry-join=consul-1 -client=0.0.0.0 -datacenter=dc1
    environment:
      - CONSUL_BIND_INTERFACE=eth0
    volumes:
      - consul-3-data:/consul/data
    networks:
      - consul-network

volumes:
  consul-1-data:
  consul-2-data:
  consul-3-data:

networks:
  consul-network:
    driver: bridge
```

---

## 分布式监控

### Prometheus 联邦/分片

```yaml
# prometheus/prometheus-federate.yml
global:
  scrape_interval: 15s

scrape_configs:
  # 从各节点 Prometheus 聚合
  - job_name: 'federate'
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{job=~".+"}'
    static_configs:
      - targets:
        - 'prometheus-node1:9090'
        - 'prometheus-node2:9090'
        - 'prometheus-node3:9090'

  # PostgreSQL 监控
  - job_name: 'postgres'
    static_configs:
      - targets: ['pg-exporter-1:9187', 'pg-exporter-2:9187']

  # Redis 集群监控
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

  # NATS 监控
  - job_name: 'nats'
    static_configs:
      - targets: ['nats-1:8222', 'nats-2:8222', 'nats-3:8222']
```

### 告警规则

```yaml
# prometheus/rules/cluster-alerts.yml
groups:
  - name: cluster
    rules:
      - alert: PostgresReplicationLag
        expr: pg_replication_lag_seconds > 30
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "PostgreSQL replication lag is high"

      - alert: RedisClusterDown
        expr: redis_cluster_state != 1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Redis cluster is down"

      - alert: NATSClusterSizeChanged
        expr: changes(nats_varz_cluster_size[5m]) > 0
        labels:
          severity: warning
        annotations:
          summary: "NATS cluster size changed"

      - alert: ServiceInstanceDown
        expr: up{job=~"gateway|provider|billing"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Service instance {{ $labels.instance }} is down"
```

---

## 完整分布式 Docker Compose

```yaml
# deploy/docker-compose.distributed.yml
version: '3.8'

services:
  # ========== Load Balancer ==========
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    depends_on:
      - gateway-service
      - new-api
    networks:
      - frontend

  # ========== Application Services ==========
  gateway-service:
    image: ghcr.io/your-org/lurus-gateway:latest
    deploy:
      replicas: 3
    environment:
      - NATS_URL=nats://nats-1:4222,nats://nats-2:4222,nats://nats-3:4222
      - REDIS_ADDRS=redis-1:7001,redis-2:7002,redis-3:7003,redis-4:7004,redis-5:7005,redis-6:7006
      - PG_DSN=postgresql://lurus:${PG_PASSWORD}@haproxy:5432/lurus_provider?sslmode=disable
    networks:
      - frontend
      - backend

  # ... other services

  # ========== PostgreSQL HA ==========
  # (参考上面 Patroni 配置)

  # ========== Redis Cluster ==========
  # (参考上面 Redis Cluster 配置)

  # ========== NATS Cluster ==========
  # (参考上面 NATS 配置)

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true
```

---

## K3S 高可用配置

如果选择 Kubernetes，K3S 支持嵌入式 etcd 高可用:

```bash
# 第一个 master 节点
curl -sfL https://get.k3s.io | sh -s - server \
  --cluster-init \
  --tls-san=<LB_IP>

# 获取 token
cat /var/lib/rancher/k3s/server/node-token

# 其他 master 节点加入
curl -sfL https://get.k3s.io | sh -s - server \
  --server https://<FIRST_SERVER>:6443 \
  --token <TOKEN> \
  --tls-san=<LB_IP>
```

详细 K3S HA 配置参考: `deploy/k3s/README.md`

---

## 迁移检查清单

### 分布式环境准备

- [ ] 规划 IP 地址和网络分段
- [ ] 准备 3+ 台服务器 (物理机/VM)
- [ ] 配置内网互通 + 防火墙规则
- [ ] 安装 Docker + Docker Compose
- [ ] 准备域名和 SSL 证书

### 组件部署顺序

1. [ ] 部署 etcd 集群 (Patroni 依赖)
2. [ ] 部署 PostgreSQL Patroni 集群
3. [ ] 部署 Redis Cluster
4. [ ] 部署 NATS Cluster
5. [ ] 部署 Consul (可选)
6. [ ] 部署应用服务
7. [ ] 配置负载均衡
8. [ ] 配置监控告警
9. [ ] 压力测试
10. [ ] DNS 切换

---

*Generated by Claude Code | 2026-01-08*
