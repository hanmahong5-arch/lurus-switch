# Lurus Switch 多节点部署指南

> **文档版本**: v1.0
> **更新日期**: 2026-01-11
> **适用场景**: 个人/小团队低成本生产部署

---

## 目录

1. [资源规划](#一资源规划)
2. [架构设计](#二架构设计)
3. [服务器选购建议](#三服务器选购建议薅羊毛指南)
4. [系统选择：Windows vs Ubuntu](#四系统选择windows-vs-ubuntu)
5. [网络配置](#五网络配置)
6. [Docker Compose 配置](#六docker-compose-配置)
7. [部署步骤](#七部署步骤)
8. [运维管理](#八运维管理)
9. [备份与恢复](#九备份与恢复)
10. [故障排查](#十故障排查)
11. [扩展路径](#十一扩展路径)

---

## 一、资源规划

### 1.1 当前资源清单

| 编号 | 配置 | 带宽/流量 | 公网IP | 系统 | 状态 | 建议角色 |
|------|------|----------|--------|------|------|---------|
| **A** | 8C / 64GB | 100M / 30G月 | 有 | Windows Server | 6月到期 | 微服务层 |
| **B** | 8C / 64GB | 100M / 30G月 | 有 | Windows Server | 正常 | 数据层 |
| **C** | 4C / 8GB | 5M / 无限流量 | 有 | - | 正常 | **入口网关** |
| **D** | ?C / 32GB / 4TB | 无 | **无** | Windows | 本地 | 备份存储 |
| **E** | 2C / 2GB | 5M / 无限流量 | 有 | - | 正常 | 备用/监控 |

### 1.2 资源特点分析

| 资源 | 优势 | 劣势 | 最佳用途 |
|------|------|------|---------|
| A/B | 超大内存(64GB)、高配置 | 流量限制(30G/月) | 内部服务、数据存储 |
| C | 无限流量、有公网IP | 配置较低(8GB) | **入口网关** |
| D | 超大存储(4TB)、大内存 | 无公网IP | 备份、冷存储 |
| E | 无限流量、有公网IP | 配置最低(2GB) | 监控告警、备用入口 |

### 1.3 流量消耗估算

| 场景 | 日流量 | 月流量 | 说明 |
|------|--------|--------|------|
| Claude Code 轻度使用 | 100MB | 3GB | 每天几十次对话 |
| Claude Code 中度使用 | 500MB | 15GB | 每天上百次对话 |
| Claude Code 重度使用 | 2GB | 60GB | 团队多人使用 |
| 内部服务通信 | 50MB | 1.5GB | 微服务间调用 |

**结论**：A/B 的 30G/月流量不够做入口，必须用 C 的无限流量做入口。

---

## 二、架构设计

### 2.1 整体架构图

```
                                    互联网
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         服务器 C (入口网关)                                    │
│                         4C / 8GB / 5M 无限流量 / Ubuntu                       │
│                                                                              │
│   ┌────────────┐    ┌──────────────────────────────────────────────────┐    │
│   │   Caddy    │───▶│  Gateway Service (x2) + NEW-API (x2)             │    │
│   │  :80/443   │    │  + lurus-portal (前端)                            │    │
│   │ 自动HTTPS  │    │  所有外部请求入口                                  │    │
│   └────────────┘    └──────────────────────────────────────────────────┘    │
│                                                                              │
│   ┌────────────────────────────────────────────────────────────────────┐    │
│   │  WireGuard VPN Hub (10.0.0.1) + frp server (:7000)                 │    │
│   └────────────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────┬───────────────────────────────────────┘
                                       │
            ┌──────────────────────────┼──────────────────────────┐
            │ WireGuard VPN            │                          │ WireGuard VPN
            ▼ (10.0.0.2)               ▼ (10.0.0.3)               ▼ (10.0.0.5)
┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐
│ 服务器 A (微服务层)    │  │ 服务器 B (数据层)      │  │ 服务器 E (监控告警)   │
│ 8C/64GB / Ubuntu      │  │ 8C/64GB / Ubuntu      │  │ 2C/2GB / Ubuntu       │
│                       │  │                       │  │                       │
│ ┌───────────────────┐ │  │ ┌───────────────────┐ │  │ ┌───────────────────┐ │
│ │  Provider Service │ │  │ │    PostgreSQL     │ │  │ │   Alertmanager    │ │
│ │  Billing Service  │ │  │ │    (Primary)      │ │  │ │                   │ │
│ │  Log Service      │ │  │ └───────────────────┘ │  │ └───────────────────┘ │
│ │  Sync Service     │ │  │                       │  │                       │
│ └───────────────────┘ │  │ ┌───────────────────┐ │  │ 备用功能:             │
│                       │  │ │      Redis        │ │  │ - 备用入口网关        │
│ ┌───────────────────┐ │  │ │   (Cache/Queue)   │ │  │ - 健康检查探针        │
│ │    Prometheus     │ │  │ └───────────────────┘ │  │ - 外部监控            │
│ │    Grafana        │ │  │                       │  │                       │
│ │    Jaeger         │ │  │ ┌───────────────────┐ │  └───────────────────────┘
│ └───────────────────┘ │  │ │       NATS        │ │
└───────────────────────┘  │ │    JetStream      │ │
                           │ └───────────────────┘ │
                           │                       │
                           │ ┌───────────────────┐ │
                           │ │    ClickHouse     │ │
                           │ │   (Log OLAP)      │ │
                           │ └───────────────────┘ │
                           └───────────────────────┘
                                       │
                                       │ frp tunnel (TCP)
                                       ▼
                           ┌───────────────────────┐
                           │ 本地机器 D (备份层)    │
                           │ 32GB / 4TB / Windows  │
                           │                       │
                           │ ┌───────────────────┐ │
                           │ │ PostgreSQL Replica│ │
                           │ │   (只读备份)       │ │
                           │ └───────────────────┘ │
                           │                       │
                           │ ┌───────────────────┐ │
                           │ │   4TB 冷存储       │ │
                           │ │ - 数据库备份       │ │
                           │ │ - 日志归档         │ │
                           │ │ - 历史数据         │ │
                           │ └───────────────────┘ │
                           └───────────────────────┘
```

### 2.2 服务分配详情

| 节点 | IP (VPN) | 运行服务 | 内存占用 | 磁盘占用 |
|------|----------|---------|---------|---------|
| **C** | 10.0.0.1 | Caddy, Gateway(x2), NEW-API(x2), Portal, frps, WireGuard | ~4GB | ~10GB |
| **A** | 10.0.0.2 | Provider, Billing, Log, Sync, Prometheus, Grafana, Jaeger | ~8GB | ~30GB |
| **B** | 10.0.0.3 | PostgreSQL, Redis, NATS, ClickHouse | ~12GB | ~100GB |
| **D** | 10.0.0.4 | PostgreSQL-replica, 备份脚本, frpc | ~4GB | ~500GB+ |
| **E** | 10.0.0.5 | Alertmanager, (备用网关) | ~1GB | ~5GB |

### 2.3 数据流向

```
用户请求流:
  用户 → C(Caddy) → C(Gateway/NEW-API) → A/B(微服务/数据库) → 返回

内部服务通信:
  A(微服务) ←→ B(数据库) via WireGuard VPN
  A(微服务) ←→ B(NATS) via WireGuard VPN

备份数据流:
  B(PostgreSQL) → D(PostgreSQL-replica) via frp tunnel
  B(数据) → D(冷存储) 定时备份

监控数据流:
  所有节点 → A(Prometheus) → A(Grafana)
  A(Alertmanager规则) → E(Alertmanager) → 告警通知
```

---

## 三、服务器选购建议（薅羊毛指南）

### 3.1 国内云服务商新用户优惠

| 平台 | 优惠活动 | 推荐配置 | 参考价格 | 有效期 | 链接 |
|------|---------|---------|---------|--------|------|
| **阿里云** | 新用户专享 | 2C2G 轻量 | ¥99/年 | 1年 | cloud.aliyun.com |
| **阿里云** | 新用户专享 | 2C4G 轻量 | ¥199/年 | 1年 | cloud.aliyun.com |
| **腾讯云** | 新人专区 | 2C2G 轻量 | ¥99/年 | 1年 | cloud.tencent.com |
| **腾讯云** | 新人专区 | 2C4G 轻量 | ¥199/年 | 1年 | cloud.tencent.com |
| **华为云** | 新用户 | 2C4G ECS | ¥89/年 | 1年 | huaweicloud.com |
| **京东云** | 新人优惠 | 2C4G | ¥99/年 | 1年 | jdcloud.com |
| **百度云** | 新用户 | 2C4G | ¥99/年 | 1年 | cloud.baidu.com |
| **天翼云** | 新人礼包 | 2C4G | ¥88/年 | 1年 | ctyun.cn |
| **移动云** | 新用户 | 2C4G | ¥99/年 | 1年 | ecloud.10086.cn |

### 3.2 海外云服务商

| 平台 | 优惠活动 | 推荐配置 | 参考价格 | 特点 |
|------|---------|---------|---------|------|
| **Vultr** | $100 免费额度 | 2C4G | $24/月 | 按小时计费，可随时删除 |
| **DigitalOcean** | $200 免费额度 | 2C4G | $24/月 | 60天有效 |
| **Linode** | $100 免费额度 | 2C4G | $24/月 | 新用户 |
| **Hetzner** | 无 | CX22 (2C4G) | €4.5/月 | 性价比最高 |
| **Oracle Cloud** | 永久免费 | 4C24G ARM | 免费 | 抢不到，但值得尝试 |
| **Google Cloud** | $300 免费额度 | e2-medium | $33/月 | 90天有效 |
| **AWS** | 12个月免费 | t2.micro | 免费 | 1C1G，配置低 |

### 3.3 薅羊毛策略

#### 方案一：多平台新用户（推荐）

```
第1年: 阿里云新用户 (¥99) + 腾讯云新用户 (¥99) = ¥198
第2年: 华为云新用户 (¥89) + 京东云新用户 (¥99) = ¥188
第3年: 百度云新用户 (¥99) + 天翼云新用户 (¥88) = ¥187
...循环使用不同平台
```

#### 方案二：海外免费额度

```
Vultr $100 (约4个月免费)
  ↓ 用完后
DigitalOcean $200 (约8个月免费)
  ↓ 用完后
Google Cloud $300 (约9个月免费)
  = 约21个月免费使用
```

#### 方案三：组合策略（最优）

```
入口网关 C: 国内云 (低延迟) - ¥99/年
数据层 B: 国内云 (低延迟) - ¥99/年
微服务 A: 海外云 (免费额度) - 免费
备用 E: 海外云 (免费额度) - 免费

年成本: ¥198 + 电费 ≈ ¥250/年
```

### 3.4 选购注意事项

1. **带宽选择**
   - 入口节点：选择**无限流量**或高流量套餐
   - 内部节点：流量限制无所谓，走VPN内网

2. **地域选择**
   - 国内用户：选择国内节点（低延迟）
   - 海外用户：选择对应地区节点
   - 入口和数据节点最好同地域（减少延迟）

3. **系统选择**
   - 推荐 Ubuntu 22.04 LTS
   - 避免 CentOS（已停止维护）

4. **续费陷阱**
   - 新用户价格通常只限首年
   - 续费价格可能是原价的3-5倍
   - 策略：到期前迁移到新平台

---

## 四、系统选择：Windows vs Ubuntu

### 4.1 对比分析

| 维度 | Windows Server | Ubuntu Server |
|------|---------------|---------------|
| **Docker 支持** | WSL2/Hyper-V，有坑 | 原生支持，完美 |
| **资源占用** | 2-4GB 系统占用 | 200-500MB 系统占用 |
| **远程管理** | RDP (图形界面) | SSH (命令行) |
| **自动化脚本** | PowerShell | Bash (更成熟) |
| **社区支持** | 较少 | 非常丰富 |
| **安全更新** | 需重启 | 可热更新 |
| **授权成本** | 需要 License | 免费 |
| **Claude Code 兼容** | 完全支持 | 完全支持 |

### 4.2 建议

| 节点 | 当前系统 | 建议系统 | 迁移优先级 |
|------|---------|---------|-----------|
| **A** | Windows Server | **Ubuntu 22.04** | 高（6月到期正好切换） |
| **B** | Windows Server | **Ubuntu 22.04** | 中（数据层，谨慎迁移） |
| **C** | - | **Ubuntu 22.04** | - |
| **D** | Windows | 保持 Windows | 低（本地机器，兼顾日常使用） |
| **E** | - | **Ubuntu 22.04** | - |

### 4.3 Windows 到 Ubuntu 迁移步骤

#### 迁移 A 节点（6月到期时）

```bash
# 1. 在新 Ubuntu 服务器上准备环境
sudo apt update && sudo apt upgrade -y
sudo apt install -y docker.io docker-compose-v2 wireguard

# 2. 配置 WireGuard VPN
sudo cp wg0.conf /etc/wireguard/
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0

# 3. 克隆代码
git clone https://github.com/your-org/lurus-switch.git /opt/lurus
cd /opt/lurus

# 4. 启动服务
docker compose -f deploy/distributed/docker-compose.services.yaml up -d

# 5. 验证服务
curl http://10.0.0.2:18101/health  # Provider Service
curl http://10.0.0.2:3000          # Grafana

# 6. 更新 DNS/反向代理指向新 IP
# 7. 关闭旧 Windows 服务器
```

#### 迁移 B 节点（数据层，需谨慎）

```bash
# 1. 在新服务器准备好后，先做全量备份
ssh old-B "docker exec postgres pg_dumpall -U lurus > /tmp/full_backup.sql"
scp old-B:/tmp/full_backup.sql .

# 2. 在新服务器启动数据库
docker compose -f deploy/distributed/docker-compose.data.yaml up -d postgres

# 3. 导入数据
cat full_backup.sql | docker exec -i postgres psql -U lurus

# 4. 验证数据完整性
docker exec postgres psql -U lurus -c "SELECT count(*) FROM users;"

# 5. 切换服务指向新数据库
# 更新其他节点的环境变量 POSTGRES_HOST

# 6. 启动其他数据服务
docker compose -f deploy/distributed/docker-compose.data.yaml up -d

# 7. 关闭旧服务器
```

### 4.4 保留 Windows 的场景

- **本地机器 D**：建议保留 Windows
  - 可能需要日常使用（浏览器、Office等）
  - Docker Desktop for Windows 也能跑备份服务
  - 4TB 硬盘可以同时存储其他文件

---

## 五、网络配置

### 5.1 WireGuard VPN 配置

#### 服务器 C (Hub节点) - /etc/wireguard/wg0.conf

```ini
[Interface]
Address = 10.0.0.1/24
ListenPort = 51820
PrivateKey = <C_PRIVATE_KEY>

# 开启 IP 转发
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# 服务器 A
[Peer]
PublicKey = <A_PUBLIC_KEY>
AllowedIPs = 10.0.0.2/32

# 服务器 B
[Peer]
PublicKey = <B_PUBLIC_KEY>
AllowedIPs = 10.0.0.3/32

# 服务器 E
[Peer]
PublicKey = <E_PUBLIC_KEY>
AllowedIPs = 10.0.0.5/32
```

#### 服务器 A - /etc/wireguard/wg0.conf

```ini
[Interface]
Address = 10.0.0.2/24
PrivateKey = <A_PRIVATE_KEY>

[Peer]
PublicKey = <C_PUBLIC_KEY>
Endpoint = <C_PUBLIC_IP>:51820
AllowedIPs = 10.0.0.0/24
PersistentKeepalive = 25
```

#### 服务器 B - /etc/wireguard/wg0.conf

```ini
[Interface]
Address = 10.0.0.3/24
PrivateKey = <B_PRIVATE_KEY>

[Peer]
PublicKey = <C_PUBLIC_KEY>
Endpoint = <C_PUBLIC_IP>:51820
AllowedIPs = 10.0.0.0/24
PersistentKeepalive = 25
```

#### 服务器 E - /etc/wireguard/wg0.conf

```ini
[Interface]
Address = 10.0.0.5/24
PrivateKey = <E_PRIVATE_KEY>

[Peer]
PublicKey = <C_PUBLIC_KEY>
Endpoint = <C_PUBLIC_IP>:51820
AllowedIPs = 10.0.0.0/24
PersistentKeepalive = 25
```

#### 生成密钥对

```bash
# 在每台服务器上执行
wg genkey | tee privatekey | wg pubkey > publickey
cat privatekey  # 填入 PrivateKey
cat publickey   # 分享给其他节点作为 PublicKey
```

#### 启动 WireGuard

```bash
# Ubuntu
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0

# 验证连接
ping 10.0.0.1  # 从任意节点 ping Hub
```

### 5.2 frp 配置（连接本地机器 D）

#### 服务器 C - frps.toml

```toml
bindPort = 7000
auth.token = "your-secure-frp-token-here"

webServer.addr = "127.0.0.1"
webServer.port = 7500
webServer.user = "admin"
webServer.password = "admin-password"
```

#### 本地机器 D - frpc.toml

```toml
serverAddr = "<C_PUBLIC_IP>"
serverPort = 7000
auth.token = "your-secure-frp-token-here"

[[proxies]]
name = "postgres-replica"
type = "tcp"
localIP = "127.0.0.1"
localPort = 5432
remotePort = 15432

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 10022
```

### 5.3 防火墙配置

#### 服务器 C（入口）

```bash
# UFW 配置
sudo ufw allow 80/tcp      # HTTP
sudo ufw allow 443/tcp     # HTTPS
sudo ufw allow 51820/udp   # WireGuard
sudo ufw allow 7000/tcp    # frp server
sudo ufw enable
```

#### 服务器 A/B/E（内部）

```bash
# 只允许 VPN 网段访问
sudo ufw allow from 10.0.0.0/24
sudo ufw allow 51820/udp   # WireGuard
sudo ufw enable
```

---

## 六、Docker Compose 配置

### 6.1 目录结构

```
deploy/distributed/
├── docker-compose.gateway.yaml    # C 节点
├── docker-compose.data.yaml       # B 节点
├── docker-compose.services.yaml   # A 节点
├── docker-compose.monitor.yaml    # E 节点
├── docker-compose.backup.yaml     # D 节点
├── .env.example                   # 环境变量模板
├── caddy/
│   └── Caddyfile
├── wireguard/
│   ├── wg0-hub.conf.example
│   └── wg0-peer.conf.example
├── frp/
│   ├── frps.toml
│   └── frpc.toml
└── scripts/
    ├── deploy-all.ps1
    ├── deploy-all.sh
    └── backup.sh
```

### 6.2 服务器 C: docker-compose.gateway.yaml

```yaml
version: "3.8"

services:
  caddy:
    image: caddy:2-alpine
    container_name: caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - lurus-net

  gateway-1:
    image: ghcr.io/your-org/gateway-service:latest
    container_name: gateway-1
    restart: unless-stopped
    environment:
      - SERVER_PORT=18100
      - POSTGRES_HOST=10.0.0.3
      - POSTGRES_PORT=5432
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=lurus
      - REDIS_HOST=10.0.0.3
      - REDIS_PORT=6379
      - NATS_URL=nats://10.0.0.3:4222
      - JAEGER_ENDPOINT=http://10.0.0.2:4317
      - LOG_LEVEL=info
    networks:
      - lurus-net
    extra_hosts:
      - "host.docker.internal:host-gateway"

  gateway-2:
    image: ghcr.io/your-org/gateway-service:latest
    container_name: gateway-2
    restart: unless-stopped
    environment:
      - SERVER_PORT=18100
      - POSTGRES_HOST=10.0.0.3
      - POSTGRES_PORT=5432
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=lurus
      - REDIS_HOST=10.0.0.3
      - REDIS_PORT=6379
      - NATS_URL=nats://10.0.0.3:4222
      - JAEGER_ENDPOINT=http://10.0.0.2:4317
      - LOG_LEVEL=info
    networks:
      - lurus-net
    extra_hosts:
      - "host.docker.internal:host-gateway"

  new-api-1:
    image: ghcr.io/your-org/new-api:latest
    container_name: new-api-1
    restart: unless-stopped
    environment:
      - SQL_DSN=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@10.0.0.3:5432/new_api
      - REDIS_CONN_STRING=redis://10.0.0.3:6379
      - SESSION_SECRET=${SESSION_SECRET}
    networks:
      - lurus-net

  new-api-2:
    image: ghcr.io/your-org/new-api:latest
    container_name: new-api-2
    restart: unless-stopped
    environment:
      - SQL_DSN=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@10.0.0.3:5432/new_api
      - REDIS_CONN_STRING=redis://10.0.0.3:6379
      - SESSION_SECRET=${SESSION_SECRET}
    networks:
      - lurus-net

  lurus-portal:
    image: ghcr.io/your-org/lurus-portal:latest
    container_name: lurus-portal
    restart: unless-stopped
    networks:
      - lurus-net

  frps:
    image: snowdreamtech/frps:latest
    container_name: frps
    restart: unless-stopped
    ports:
      - "7000:7000"
      - "7500:7500"
      - "15432:15432"
      - "10022:10022"
    volumes:
      - ./frp/frps.toml:/etc/frp/frps.toml:ro
    networks:
      - lurus-net

networks:
  lurus-net:
    driver: bridge

volumes:
  caddy_data:
  caddy_config:
```

### 6.3 Caddyfile

```caddyfile
# AI API Gateway
ai.{$DOMAIN} {
    # Claude Code / Codex API
    @claude path /v1/messages*
    handle @claude {
        reverse_proxy gateway-1:18100 gateway-2:18100 {
            lb_policy round_robin
            health_uri /health
            health_interval 10s
        }
    }

    @codex path /responses*
    handle @codex {
        reverse_proxy gateway-1:18100 gateway-2:18100 {
            lb_policy round_robin
        }
    }

    @openai path /v1/chat/completions*
    handle @openai {
        reverse_proxy gateway-1:18100 gateway-2:18100 {
            lb_policy round_robin
        }
    }

    @gemini path /v1beta/*
    handle @gemini {
        reverse_proxy gateway-1:18100 gateway-2:18100 {
            lb_policy round_robin
        }
    }

    # 默认到前端
    handle {
        reverse_proxy lurus-portal:80
    }
}

# NEW-API 管理后台
api.{$DOMAIN} {
    reverse_proxy new-api-1:3000 new-api-2:3000 {
        lb_policy round_robin
        health_uri /api/status
        health_interval 10s
    }
}

# Grafana 监控 (反代到 A 节点)
grafana.{$DOMAIN} {
    reverse_proxy 10.0.0.2:3000
}

# 可选: Jaeger UI
jaeger.{$DOMAIN} {
    reverse_proxy 10.0.0.2:16686
}
```

### 6.4 服务器 B: docker-compose.data.yaml

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    container_name: postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: lurus
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init-databases.sql:/docker-entrypoint-initdb.d/init.sql:ro
    ports:
      - "10.0.0.3:5432:5432"
    networks:
      - lurus-net
    command: >
      postgres
        -c max_connections=200
        -c shared_buffers=2GB
        -c effective_cache_size=6GB
        -c maintenance_work_mem=512MB
        -c checkpoint_completion_target=0.9
        -c wal_buffers=16MB
        -c default_statistics_target=100
        -c random_page_cost=1.1
        -c effective_io_concurrency=200

  redis:
    image: redis:7-alpine
    container_name: redis
    restart: unless-stopped
    command: >
      redis-server
        --appendonly yes
        --maxmemory 4gb
        --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    ports:
      - "10.0.0.3:6379:6379"
    networks:
      - lurus-net

  nats:
    image: nats:2.10-alpine
    container_name: nats
    restart: unless-stopped
    command: >
      --jetstream
      --store_dir=/data
      --max_mem_store=2G
      --max_file_store=10G
    volumes:
      - nats_data:/data
    ports:
      - "10.0.0.3:4222:4222"
      - "10.0.0.3:8222:8222"
    networks:
      - lurus-net

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    container_name: clickhouse
    restart: unless-stopped
    environment:
      CLICKHOUSE_USER: ${CLICKHOUSE_USER}
      CLICKHOUSE_PASSWORD: ${CLICKHOUSE_PASSWORD}
      CLICKHOUSE_DB: lurus_logs
    volumes:
      - clickhouse_data:/var/lib/clickhouse
      - ./clickhouse/config.xml:/etc/clickhouse-server/config.d/config.xml:ro
    ports:
      - "10.0.0.3:8123:8123"
      - "10.0.0.3:9000:9000"
    networks:
      - lurus-net
    ulimits:
      nofile:
        soft: 262144
        hard: 262144

networks:
  lurus-net:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
  nats_data:
  clickhouse_data:
```

### 6.5 服务器 A: docker-compose.services.yaml

```yaml
version: "3.8"

services:
  provider-service:
    image: ghcr.io/your-org/provider-service:latest
    container_name: provider-service
    restart: unless-stopped
    environment:
      - SERVER_PORT=18101
      - DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@10.0.0.3:5432/provider
      - REDIS_URL=redis://10.0.0.3:6379
      - NATS_URL=nats://10.0.0.3:4222
      - JAEGER_ENDPOINT=jaeger:4317
    ports:
      - "10.0.0.2:18101:18101"
    networks:
      - lurus-net

  billing-service:
    image: ghcr.io/your-org/billing-service:latest
    container_name: billing-service
    restart: unless-stopped
    environment:
      - SERVER_PORT=18103
      - DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@10.0.0.3:5432/billing
      - REDIS_URL=redis://10.0.0.3:6379
      - NATS_URL=nats://10.0.0.3:4222
      - JAEGER_ENDPOINT=jaeger:4317
    ports:
      - "10.0.0.2:18103:18103"
    networks:
      - lurus-net

  log-service:
    image: ghcr.io/your-org/log-service:latest
    container_name: log-service
    restart: unless-stopped
    environment:
      - SERVER_PORT=18102
      - CLICKHOUSE_HOST=10.0.0.3
      - CLICKHOUSE_PORT=9000
      - CLICKHOUSE_USER=${CLICKHOUSE_USER}
      - CLICKHOUSE_PASSWORD=${CLICKHOUSE_PASSWORD}
      - NATS_URL=nats://10.0.0.3:4222
      - JAEGER_ENDPOINT=jaeger:4317
    ports:
      - "10.0.0.2:18102:18102"
    networks:
      - lurus-net

  sync-service:
    image: ghcr.io/your-org/sync-service:latest
    container_name: sync-service
    restart: unless-stopped
    environment:
      - SERVER_PORT=8081
      - DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@10.0.0.3:5432/sync
      - REDIS_URL=redis://10.0.0.3:6379
      - NATS_URL=nats://10.0.0.3:4222
      - JAEGER_ENDPOINT=jaeger:4317
    ports:
      - "10.0.0.2:8081:8081"
    networks:
      - lurus-net

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    restart: unless-stopped
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./prometheus/rules:/etc/prometheus/rules:ro
      - prometheus_data:/prometheus
    ports:
      - "10.0.0.2:9090:9090"
    networks:
      - lurus-net

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    ports:
      - "10.0.0.2:3000:3000"
    networks:
      - lurus-net

  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: jaeger
    restart: unless-stopped
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "10.0.0.2:16686:16686"
      - "10.0.0.2:4317:4317"
      - "10.0.0.2:4318:4318"
    networks:
      - lurus-net

networks:
  lurus-net:
    driver: bridge

volumes:
  prometheus_data:
  grafana_data:
```

### 6.6 服务器 E: docker-compose.monitor.yaml

```yaml
version: "3.8"

services:
  alertmanager:
    image: prom/alertmanager:latest
    container_name: alertmanager
    restart: unless-stopped
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    volumes:
      - ./alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro
      - alertmanager_data:/alertmanager
    ports:
      - "10.0.0.5:9093:9093"
    networks:
      - lurus-net

  # 可选: 健康检查服务
  healthcheck:
    image: alpine:latest
    container_name: healthcheck
    restart: unless-stopped
    command: |
      sh -c 'while true; do
        curl -s http://10.0.0.1/health > /dev/null || echo "Gateway DOWN!"
        curl -s http://10.0.0.3:5432 > /dev/null || echo "PostgreSQL DOWN!"
        sleep 60
      done'
    networks:
      - lurus-net

networks:
  lurus-net:
    driver: bridge

volumes:
  alertmanager_data:
```

### 6.7 本地机器 D: docker-compose.backup.yaml (Windows)

```yaml
version: "3.8"

services:
  postgres-replica:
    image: postgres:16-alpine
    container_name: postgres-replica
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - D:/lurus-backup/postgres:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    command: >
      postgres
        -c hot_standby=on

  frpc:
    image: snowdreamtech/frpc:latest
    container_name: frpc
    restart: unless-stopped
    volumes:
      - ./frpc.toml:/etc/frp/frpc.toml:ro

  # 定时备份服务
  backup:
    image: postgres:16-alpine
    container_name: backup-job
    volumes:
      - D:/lurus-backup/dumps:/backups
    entrypoint: |
      sh -c 'while true; do
        pg_dump -h host.docker.internal -p 15432 -U lurus lurus > /backups/lurus_$(date +%Y%m%d_%H%M).sql
        # 保留最近30天的备份
        find /backups -name "*.sql" -mtime +30 -delete
        sleep 86400
      done'
```

### 6.8 环境变量模板 .env.example

```bash
# PostgreSQL
POSTGRES_USER=lurus
POSTGRES_PASSWORD=your-secure-postgres-password

# Redis (如果需要密码)
REDIS_PASSWORD=your-secure-redis-password

# ClickHouse
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=your-secure-clickhouse-password

# NEW-API
SESSION_SECRET=your-secure-session-secret

# Grafana
GRAFANA_PASSWORD=your-secure-grafana-password

# Domain
DOMAIN=lurus.cn

# frp
FRP_TOKEN=your-secure-frp-token
```

---

## 七、部署步骤

### 7.1 Phase 1: 准备工作

```bash
# 1. 在所有服务器上安装 Docker (Ubuntu)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

# 2. 安装 Docker Compose
sudo apt install docker-compose-v2 -y

# 3. 安装 WireGuard
sudo apt install wireguard -y

# 4. 克隆代码仓库
git clone https://github.com/your-org/lurus-switch.git /opt/lurus
cd /opt/lurus

# 5. 复制环境变量
cp deploy/distributed/.env.example deploy/distributed/.env
# 编辑 .env 填入实际值
```

### 7.2 Phase 2: 网络配置

```bash
# 在每台服务器上:

# 1. 生成 WireGuard 密钥
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey

# 2. 配置 WireGuard (根据节点角色)
sudo vim /etc/wireguard/wg0.conf
# 填入对应配置

# 3. 启动 WireGuard
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0

# 4. 验证 VPN 连接
ping 10.0.0.1  # 从任意节点 ping Hub
```

### 7.3 Phase 3: 部署数据层 (B 节点)

```bash
# 在 B 节点执行
cd /opt/lurus/deploy/distributed

# 1. 启动数据服务
docker compose -f docker-compose.data.yaml up -d

# 2. 验证服务
docker ps
docker logs postgres
docker logs redis
docker logs nats

# 3. 初始化数据库
docker exec -it postgres psql -U lurus -c "\l"
```

### 7.4 Phase 4: 部署微服务层 (A 节点)

```bash
# 在 A 节点执行
cd /opt/lurus/deploy/distributed

# 1. 验证能连接 B 节点
nc -zv 10.0.0.3 5432
nc -zv 10.0.0.3 6379

# 2. 启动微服务
docker compose -f docker-compose.services.yaml up -d

# 3. 验证服务
curl http://10.0.0.2:18101/health
curl http://10.0.0.2:3000  # Grafana
```

### 7.5 Phase 5: 部署入口网关 (C 节点)

```bash
# 在 C 节点执行
cd /opt/lurus/deploy/distributed

# 1. 验证能连接 B 节点
nc -zv 10.0.0.3 5432
nc -zv 10.0.0.3 4222

# 2. 启动网关服务
docker compose -f docker-compose.gateway.yaml up -d

# 3. 验证服务
curl http://localhost/health
curl https://ai.lurus.cn/health  # 外部访问
```

### 7.6 Phase 6: 配置域名 DNS

```
在域名注册商后台添加 A 记录:

ai.lurus.cn      → C 节点公网 IP
api.lurus.cn     → C 节点公网 IP
grafana.lurus.cn → C 节点公网 IP
```

### 7.7 Phase 7: 部署监控 (E 节点)

```bash
# 在 E 节点执行
cd /opt/lurus/deploy/distributed

docker compose -f docker-compose.monitor.yaml up -d
```

### 7.8 Phase 8: 配置备份 (D 节点)

```powershell
# 在 D 节点 (Windows) 执行
cd C:\lurus-backup

# 1. 启动 Docker Desktop
# 2. 运行备份服务
docker compose -f docker-compose.backup.yaml up -d
```

---

## 八、运维管理

### 8.1 一键部署脚本

#### deploy-all.sh (Linux/macOS)

```bash
#!/bin/bash
set -e

# 配置
GATEWAY_HOST="user@c-server"
DATA_HOST="user@b-server"
SERVICES_HOST="user@a-server"
MONITOR_HOST="user@e-server"

# 颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}>>> 推送代码到 Git...${NC}"
git push origin main

deploy_node() {
    local name=$1
    local host=$2
    local compose_file=$3

    echo -e "${YELLOW}>>> 部署 $name...${NC}"
    ssh $host "cd /opt/lurus && git pull && docker compose -f deploy/distributed/$compose_file up -d"
}

case "${1:-all}" in
    gateway)
        deploy_node "Gateway (C)" "$GATEWAY_HOST" "docker-compose.gateway.yaml"
        ;;
    data)
        deploy_node "Data (B)" "$DATA_HOST" "docker-compose.data.yaml"
        ;;
    services)
        deploy_node "Services (A)" "$SERVICES_HOST" "docker-compose.services.yaml"
        ;;
    monitor)
        deploy_node "Monitor (E)" "$MONITOR_HOST" "docker-compose.monitor.yaml"
        ;;
    all)
        deploy_node "Data (B)" "$DATA_HOST" "docker-compose.data.yaml"
        deploy_node "Services (A)" "$SERVICES_HOST" "docker-compose.services.yaml"
        deploy_node "Gateway (C)" "$GATEWAY_HOST" "docker-compose.gateway.yaml"
        deploy_node "Monitor (E)" "$MONITOR_HOST" "docker-compose.monitor.yaml"
        ;;
    *)
        echo "Usage: $0 [gateway|data|services|monitor|all]"
        exit 1
        ;;
esac

echo -e "${GREEN}>>> 部署完成!${NC}"
```

#### deploy-all.ps1 (Windows)

```powershell
param(
    [Parameter(Position=0)]
    [ValidateSet("gateway", "data", "services", "monitor", "all")]
    [string]$Target = "all"
)

$GATEWAY_HOST = "user@c-server"
$DATA_HOST = "user@b-server"
$SERVICES_HOST = "user@a-server"
$MONITOR_HOST = "user@e-server"

function Deploy-Node {
    param($Name, $Host, $ComposeFile)

    Write-Host ">>> 部署 $Name..." -ForegroundColor Yellow
    ssh $Host "cd /opt/lurus && git pull && docker compose -f deploy/distributed/$ComposeFile up -d"
}

Write-Host ">>> 推送代码到 Git..." -ForegroundColor Green
git push origin main

switch ($Target) {
    "gateway" {
        Deploy-Node "Gateway (C)" $GATEWAY_HOST "docker-compose.gateway.yaml"
    }
    "data" {
        Deploy-Node "Data (B)" $DATA_HOST "docker-compose.data.yaml"
    }
    "services" {
        Deploy-Node "Services (A)" $SERVICES_HOST "docker-compose.services.yaml"
    }
    "monitor" {
        Deploy-Node "Monitor (E)" $MONITOR_HOST "docker-compose.monitor.yaml"
    }
    "all" {
        Deploy-Node "Data (B)" $DATA_HOST "docker-compose.data.yaml"
        Deploy-Node "Services (A)" $SERVICES_HOST "docker-compose.services.yaml"
        Deploy-Node "Gateway (C)" $GATEWAY_HOST "docker-compose.gateway.yaml"
        Deploy-Node "Monitor (E)" $MONITOR_HOST "docker-compose.monitor.yaml"
    }
}

Write-Host ">>> 部署完成!" -ForegroundColor Green
```

### 8.2 日常运维命令

```bash
# ===== 状态检查 =====
# 查看所有节点状态
ssh c-server "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
ssh b-server "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
ssh a-server "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

# 查看资源使用
ssh b-server "docker stats --no-stream"

# ===== 日志查看 =====
# 查看 Gateway 日志
ssh c-server "docker logs -f --tail 100 gateway-1"

# 查看数据库日志
ssh b-server "docker logs -f --tail 100 postgres"

# 查看微服务日志
ssh a-server "docker logs -f --tail 100 billing-service"

# ===== 服务管理 =====
# 重启单个服务
ssh a-server "docker compose -f /opt/lurus/deploy/distributed/docker-compose.services.yaml restart billing-service"

# 重启所有网关服务
ssh c-server "docker compose -f /opt/lurus/deploy/distributed/docker-compose.gateway.yaml restart"

# 拉取最新镜像并重启
ssh c-server "cd /opt/lurus && docker compose -f deploy/distributed/docker-compose.gateway.yaml pull && docker compose -f deploy/distributed/docker-compose.gateway.yaml up -d"

# ===== 数据库操作 =====
# 进入 PostgreSQL
ssh b-server "docker exec -it postgres psql -U lurus"

# 执行 SQL
ssh b-server "docker exec postgres psql -U lurus -c 'SELECT count(*) FROM users;'"

# 查看数据库大小
ssh b-server "docker exec postgres psql -U lurus -c \"SELECT pg_size_pretty(pg_database_size('lurus'));\""

# ===== 网络检查 =====
# 检查 VPN 连接
ping 10.0.0.1  # Hub
ping 10.0.0.2  # A
ping 10.0.0.3  # B

# 检查端口连通性
nc -zv 10.0.0.3 5432  # PostgreSQL
nc -zv 10.0.0.3 6379  # Redis
nc -zv 10.0.0.3 4222  # NATS
```

### 8.3 Claude Code 管理方式

```
开发电脑 (运行 Claude Code)
    │
    ├─► SSH 到 C 节点: ssh c-server
    │       └─► 管理入口网关服务
    │
    ├─► SSH 到 A 节点: ssh a-server
    │       └─► 管理微服务和监控
    │
    ├─► SSH 到 B 节点: ssh b-server
    │       └─► 管理数据库服务
    │
    └─► 通过 WireGuard VPN 直连所有节点
            └─► 可以用 kubectl (如果有 K3s)
            └─► 可以用 curl 测试 API

工作流:
1. 在本地用 Claude Code 修改代码
2. git push 推送到仓库
3. 运行 ./deploy-all.sh 一键部署
4. 在 Grafana (grafana.lurus.cn) 查看监控
```

---

## 九、备份与恢复

### 9.1 备份策略

| 数据类型 | 备份频率 | 保留期 | 存储位置 |
|---------|---------|--------|---------|
| PostgreSQL 全量 | 每日 | 30天 | D 节点 |
| PostgreSQL 增量 | 实时流复制 | - | D 节点 |
| Redis | 每小时 | 7天 | B 节点本地 |
| NATS JetStream | 持久化 | 7天 | B 节点本地 |
| 配置文件 | Git | 永久 | GitHub |

### 9.2 备份脚本 (backup.sh)

```bash
#!/bin/bash
# 在 D 节点运行

BACKUP_DIR="/d/lurus-backup/dumps"
REMOTE_HOST="10.0.0.3"  # B 节点 VPN IP
DATE=$(date +%Y%m%d_%H%M%S)

# PostgreSQL 全量备份
echo ">>> 备份 PostgreSQL..."
docker exec postgres pg_dump -h $REMOTE_HOST -U lurus -d lurus > $BACKUP_DIR/lurus_$DATE.sql
docker exec postgres pg_dump -h $REMOTE_HOST -U lurus -d new_api > $BACKUP_DIR/new_api_$DATE.sql
docker exec postgres pg_dump -h $REMOTE_HOST -U lurus -d billing > $BACKUP_DIR/billing_$DATE.sql

# 压缩
gzip $BACKUP_DIR/*_$DATE.sql

# 清理30天前的备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo ">>> 备份完成: $BACKUP_DIR"
```

### 9.3 恢复步骤

```bash
# 1. 停止相关服务
ssh c-server "docker compose -f /opt/lurus/deploy/distributed/docker-compose.gateway.yaml stop"

# 2. 恢复数据库
gunzip /d/lurus-backup/dumps/lurus_20260111_120000.sql.gz
cat /d/lurus-backup/dumps/lurus_20260111_120000.sql | ssh b-server "docker exec -i postgres psql -U lurus"

# 3. 重启服务
ssh c-server "docker compose -f /opt/lurus/deploy/distributed/docker-compose.gateway.yaml start"

# 4. 验证
curl https://ai.lurus.cn/health
```

---

## 十、故障排查

### 10.1 常见问题

| 问题 | 可能原因 | 解决方案 |
|------|---------|---------|
| 无法访问 ai.lurus.cn | DNS 未生效 / Caddy 挂了 | 检查 DNS，重启 Caddy |
| 502 Bad Gateway | 后端服务挂了 | 检查 Gateway 服务状态 |
| 数据库连接失败 | VPN 断开 / PostgreSQL 挂了 | 检查 VPN，重启 PostgreSQL |
| 响应慢 | Redis 满了 / 数据库慢查询 | 清理 Redis，优化 SQL |
| 磁盘满 | 日志/数据增长 | 清理日志，扩容磁盘 |

### 10.2 排查命令

```bash
# 检查服务健康
curl -s http://localhost/health | jq

# 检查 Docker 容器
docker ps -a
docker logs --tail 50 <container_name>

# 检查资源使用
docker stats --no-stream
df -h
free -h

# 检查网络连通性
ping 10.0.0.1
nc -zv 10.0.0.3 5432
curl -I https://ai.lurus.cn

# 检查 WireGuard
sudo wg show

# 检查进程
ps aux | grep docker
netstat -tlnp
```

### 10.3 应急恢复

```bash
# 场景1: 单个服务挂了
docker restart <service_name>

# 场景2: 整个节点服务挂了
docker compose -f docker-compose.xxx.yaml down
docker compose -f docker-compose.xxx.yaml up -d

# 场景3: VPN 断开
sudo systemctl restart wg-quick@wg0

# 场景4: 磁盘满
# 清理 Docker 资源
docker system prune -af
# 清理日志
truncate -s 0 /var/lib/docker/containers/*/*-json.log

# 场景5: 需要回滚
git checkout <previous_commit>
./deploy-all.sh
```

---

## 十一、扩展路径

### 11.1 当前架构可扩展性

| 瓶颈点 | 当前配置 | 扩展方案 |
|--------|---------|---------|
| 入口带宽 | C 节点 5M | 增加 E 节点做负载均衡 |
| 数据库性能 | 单实例 PostgreSQL | 读写分离 (D 做只读副本) |
| 微服务容量 | A 节点 64GB | 已足够，无需扩展 |
| 存储容量 | B 节点 100GB | 迁移历史数据到 D 节点 |

### 11.2 下一步扩展计划

```
当前: 4 节点 Docker Compose
  ↓
Phase 2: 增加高可用
  - A 节点到期后，迁移到新服务器
  - 增加 PostgreSQL 主从复制
  - 增加 Redis 哨兵模式
  ↓
Phase 3: 迁移到 K3s (可选)
  - 当需要更复杂的编排时
  - 自动扩缩容
  - 滚动更新
```

### 11.3 成本优化建议

1. **薅羊毛循环**：每年切换云服务商，利用新用户优惠
2. **关闭闲置服务**：开发环境可以关闭 Jaeger、ClickHouse
3. **D 节点充分利用**：本地机器电费固定，多跑些服务
4. **流量监控**：监控 A/B 节点流量，避免超额

---

## 附录 A: SSH 配置

### ~/.ssh/config

```
Host c-server
    HostName <C_PUBLIC_IP>
    User ubuntu
    IdentityFile ~/.ssh/lurus_key

Host a-server
    HostName 10.0.0.2
    User ubuntu
    IdentityFile ~/.ssh/lurus_key
    ProxyJump c-server

Host b-server
    HostName 10.0.0.3
    User ubuntu
    IdentityFile ~/.ssh/lurus_key
    ProxyJump c-server

Host e-server
    HostName 10.0.0.5
    User ubuntu
    IdentityFile ~/.ssh/lurus_key
    ProxyJump c-server

Host d-local
    HostName <C_PUBLIC_IP>
    Port 10022
    User administrator
    IdentityFile ~/.ssh/lurus_key
```

---

## 附录 B: 快速参考卡片

```
┌─────────────────────────────────────────────────────────────┐
│                    Lurus Switch 快速参考                     │
├─────────────────────────────────────────────────────────────┤
│ 节点 IP:                                                     │
│   C (入口): 10.0.0.1 / <公网IP>                             │
│   A (微服务): 10.0.0.2                                       │
│   B (数据): 10.0.0.3                                         │
│   D (备份): 10.0.0.4 (via frp)                              │
│   E (监控): 10.0.0.5                                         │
├─────────────────────────────────────────────────────────────┤
│ 域名:                                                        │
│   ai.lurus.cn     → Gateway Service                         │
│   api.lurus.cn    → NEW-API                                 │
│   grafana.lurus.cn → Grafana                                │
├─────────────────────────────────────────────────────────────┤
│ 常用命令:                                                    │
│   部署全部: ./deploy-all.sh all                              │
│   部署网关: ./deploy-all.sh gateway                          │
│   查看日志: ssh c-server "docker logs -f gateway-1"          │
│   重启服务: ssh c-server "docker restart gateway-1"          │
│   备份数据: ssh d-local "./backup.sh"                        │
├─────────────────────────────────────────────────────────────┤
│ 监控面板:                                                    │
│   Grafana: https://grafana.lurus.cn (admin/<password>)      │
│   Prometheus: http://10.0.0.2:9090                          │
│   Jaeger: http://10.0.0.2:16686                             │
└─────────────────────────────────────────────────────────────┘
```

---

**文档结束**

> 下次继续时，从 [第七章 部署步骤](#七部署步骤) 开始执行。
