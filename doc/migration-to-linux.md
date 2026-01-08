# Windows → Ubuntu 迁移准备指南 / Migration to Ubuntu Guide

> 版本: v1.0 | 创建日期: 2026-01-08

本文档描述将 Lurus Switch 从 Windows Server 2019 迁移到 Ubuntu Linux 的准备工作和执行步骤。

---

## 目录

1. [迁移前检查清单](#迁移前检查清单)
2. [数据备份](#数据备份)
3. [已就绪的资源](#已就绪的资源)
4. [迁移步骤](#迁移步骤)
5. [迁移后验证](#迁移后验证)

---

## 迁移前检查清单

### ✅ 已就绪 (Ready)

| 项目 | 状态 | 说明 |
|------|------|------|
| Dockerfiles | ✅ | 所有服务已有 Linux Dockerfile |
| docker-compose.production.yml | ✅ | 完整的生产部署配置 |
| Caddyfile | ✅ | 边缘代理配置 |
| Prometheus/Grafana 配置 | ✅ | 可观测性配置 |
| 数据库初始化脚本 | ✅ | PostgreSQL/ClickHouse init SQL |
| CI/CD 流水线 | ✅ | GitHub Actions 配置 |
| K3S 部署清单 | ✅ | Kubernetes YAML (可选) |

### ⚠️ 需要备份 (Backup Required)

| 数据 | 位置 | 备份方式 |
|------|------|---------|
| PostgreSQL 数据 | D:\data\postgresql | pg_dump |
| Redis 数据 | D:\tools\redis\data | RDB 文件 |
| NEW-API 数据 | D:\services\new-api\data | 文件复制 |
| 静态站点 | D:\sites\ | 文件复制 |
| SSL 证书 | Caddy 自动管理 | 无需备份 |

### 📝 需要记录 (Document Required)

| 配置项 | 当前值 |
|--------|--------|
| PostgreSQL 用户 | lurus |
| PostgreSQL 密码 | lurus_dev_2024 |
| PostgreSQL 超级用户密码 | postgres |
| ACME Email | admin@lurus.cn |
| 域名列表 | api.lurus.cn, ai.lurus.cn, lurus.cn, platform.lurus.cn, portal.lurus.cn |

---

## 数据备份

### 1. PostgreSQL 数据库备份

```powershell
# 在 Windows 上执行
cd D:\PostgreSQL\bin

# 备份所有数据库
pg_dump -U postgres -h localhost lurus_provider > D:\backup\lurus_provider.sql
pg_dump -U postgres -h localhost lurus_billing > D:\backup\lurus_billing.sql
pg_dump -U postgres -h localhost lurus_sync > D:\backup\lurus_sync.sql
pg_dump -U postgres -h localhost lurus_subscription > D:\backup\lurus_subscription.sql
pg_dump -U postgres -h localhost new_api > D:\backup\new_api.sql

# 或者一次性备份所有
pg_dumpall -U postgres -h localhost > D:\backup\postgresql_all.sql
```

### 2. Redis 数据备份

```powershell
# 触发 RDB 快照
redis-cli -a lurus_dev_2024 BGSAVE

# 复制 RDB 文件
Copy-Item D:\tools\redis\data\dump.rdb D:\backup\redis_dump.rdb
```

### 3. 静态文件备份

```powershell
# 压缩静态站点
Compress-Archive -Path D:\sites\* -DestinationPath D:\backup\sites.zip

# 压缩 lurus-portal
Compress-Archive -Path D:\tools\lurus-switch\lurus-portal\.output\public\* -DestinationPath D:\backup\portal.zip
```

### 4. 配置文件备份

```powershell
# 创建配置备份目录
New-Item -ItemType Directory -Path D:\backup\configs -Force

# 复制关键配置
Copy-Item D:\services\*\configs\*.yaml D:\backup\configs\ -Recurse
Copy-Item D:\services\caddy\Caddyfile D:\backup\configs\
Copy-Item D:\services\nats\nats-server.conf D:\backup\configs\
```

### 5. 上传备份到安全位置

```powershell
# 压缩所有备份
Compress-Archive -Path D:\backup\* -DestinationPath D:\lurus-backup-20260108.zip

# 上传到 OSS 或其他存储 (示例)
# aliyun oss cp D:\lurus-backup-20260108.zip oss://your-bucket/backups/
```

---

## 已就绪的资源

### Docker 配置文件位置

```
lurus-switch/
├── deploy/
│   ├── docker-compose.production.yml   # 主部署配置
│   ├── docker-compose.dev.yaml         # 开发环境
│   ├── caddy/
│   │   └── Caddyfile                   # 边缘代理配置
│   ├── prometheus/
│   │   └── prometheus.yml              # 监控配置
│   ├── grafana/
│   │   └── provisioning/               # 仪表盘配置
│   ├── postgres/
│   │   └── init-databases.sql          # 数据库初始化
│   └── nats/
│       └── nats-server.conf            # NATS 配置
├── gateway-service/
│   └── Dockerfile                      # Gateway 镜像
├── provider-service/
│   └── Dockerfile                      # Provider 镜像
├── billing-service/
│   └── Dockerfile                      # Billing 镜像
├── log-service/
│   └── Dockerfile                      # Log 镜像
└── new-api/
    └── Dockerfile                      # NEW-API 镜像
```

### 环境变量模板

创建 `deploy/.env` 文件:

```env
# Database
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_dev_2024

# Redis
REDIS_PASSWORD=lurus_dev_2024

# ClickHouse
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=lurus_dev_2024

# Security
SESSION_SECRET=your-session-secret-here
JWT_SECRET=your-jwt-secret-here

# Grafana
GRAFANA_PASSWORD=admin

# ACME (Let's Encrypt)
ACME_EMAIL=admin@lurus.cn
```

---

## 迁移步骤

### Phase 1: Ubuntu 系统准备

```bash
# 1. 安装 Ubuntu Server 22.04 LTS 或 24.04 LTS

# 2. 更新系统
sudo apt update && sudo apt upgrade -y

# 3. 安装 Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER

# 4. 安装 Docker Compose
sudo apt install docker-compose-plugin -y

# 5. 验证安装
docker --version
docker compose version
```

### Phase 2: 代码和配置部署

```bash
# 1. 克隆代码库
cd /opt
sudo git clone https://github.com/your-org/lurus-switch.git
sudo chown -R $USER:$USER lurus-switch
cd lurus-switch

# 2. 创建必要目录
mkdir -p sites/ailurus sites/platform sites/update
mkdir -p deploy/logs/caddy

# 3. 上传备份文件并解压
# scp user@old-server:/backup/lurus-backup-20260108.zip .
unzip lurus-backup-20260108.zip -d /tmp/backup

# 4. 恢复静态文件
unzip /tmp/backup/sites.zip -d sites/
unzip /tmp/backup/portal.zip -d lurus-portal/.output/public/

# 5. 创建 .env 文件
cp deploy/.env.example deploy/.env
nano deploy/.env  # 编辑配置
```

### Phase 3: 启动服务

```bash
cd /opt/lurus-switch/deploy

# 1. 启动基础设施 (先启动数据库)
docker compose -f docker-compose.production.yml up -d postgres redis nats

# 2. 等待数据库就绪
sleep 30

# 3. 恢复 PostgreSQL 数据
docker exec -i lurus-postgres psql -U lurus < /tmp/backup/postgresql_all.sql

# 4. 启动所有服务
docker compose -f docker-compose.production.yml up -d

# 5. 查看日志
docker compose -f docker-compose.production.yml logs -f
```

### Phase 4: DNS 切换

```bash
# 1. 验证服务正常
curl -v http://localhost/health
curl -v http://localhost:18100/health

# 2. 更新 DNS 记录指向新服务器 IP
# api.lurus.cn    A  <新服务器IP>
# ai.lurus.cn     A  <新服务器IP>
# lurus.cn        A  <新服务器IP>
# ...

# 3. 等待 DNS 生效 (TTL)
# 4. Caddy 会自动申请 SSL 证书
```

---

## 迁移后验证

### 1. 服务健康检查

```bash
# 检查所有容器状态
docker compose -f docker-compose.production.yml ps

# 预期输出: 所有服务 healthy
```

### 2. API 端点验证

```bash
# Gateway
curl https://ai.lurus.cn/health
# 预期: {"status":"healthy"}

# NEW-API
curl https://api.lurus.cn/api/status
# 预期: {"success":true,...}

# Billing Sync API
curl https://api.lurus.cn/billing/api/v1/billing/sync/test-user
# 预期: {"user_id":"test-user",...}
```

### 3. 数据完整性验证

```bash
# 连接 PostgreSQL
docker exec -it lurus-postgres psql -U lurus -d lurus_billing

# 检查数据
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM usage_records;
```

### 4. 可观测性验证

```bash
# Grafana
curl https://grafana.lurus.cn/api/health
# 打开浏览器访问 https://grafana.lurus.cn

# Prometheus
docker exec lurus-prometheus wget -qO- http://localhost:9090/-/healthy
```

### 5. SSL 证书验证

```bash
# 检查证书
echo | openssl s_client -connect api.lurus.cn:443 2>/dev/null | openssl x509 -noout -dates
```

---

## 常用运维命令

### 服务管理

```bash
cd /opt/lurus-switch/deploy

# 启动所有服务
docker compose -f docker-compose.production.yml up -d

# 停止所有服务
docker compose -f docker-compose.production.yml down

# 重启单个服务
docker compose -f docker-compose.production.yml restart gateway-service

# 查看日志
docker compose -f docker-compose.production.yml logs -f gateway-service

# 进入容器
docker exec -it lurus-gateway sh
```

### 更新部署

```bash
cd /opt/lurus-switch

# 拉取最新代码
git pull

# 重新构建并部署
docker compose -f deploy/docker-compose.production.yml build
docker compose -f deploy/docker-compose.production.yml up -d
```

### 数据库备份 (Linux)

```bash
# PostgreSQL 备份
docker exec lurus-postgres pg_dumpall -U lurus > /backup/postgres_$(date +%Y%m%d).sql

# 设置 cron 定时备份
# 0 2 * * * /opt/lurus-switch/scripts/backup.sh
```

---

## 回滚方案

如果迁移出现问题:

1. **DNS 回滚**: 将 DNS 记录改回旧服务器 IP
2. **启动旧服务**: 在 Windows 服务器执行 `D:\services\start-all.ps1`
3. **数据恢复**: 如有必要，从最近备份恢复

---

## 迁移检查清单

- [ ] PostgreSQL 数据已备份
- [ ] Redis 数据已备份
- [ ] 静态文件已备份
- [ ] 配置文件已记录
- [ ] .env 文件已准备
- [ ] Ubuntu 系统已安装
- [ ] Docker 已安装
- [ ] 代码已克隆
- [ ] 数据已恢复
- [ ] 服务已启动
- [ ] 健康检查通过
- [ ] DNS 已切换
- [ ] SSL 证书已生效
- [ ] 监控正常
- [ ] 旧服务器已关闭

---

*Generated by Claude Code | 2026-01-08*
