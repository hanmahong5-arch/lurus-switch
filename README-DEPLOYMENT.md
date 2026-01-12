# Lurus Switch 部署指南总结

本项目的微服务已准备好部署到服务器 **115.190.239.146**。

## 快速开始 (3步完成部署)

### 第 1 步:配置 SSH 密钥 ⚡

在您的 **Git Bash** 终端中执行以下命令:

```bash
cat ~/.ssh/id_rsa.pub | ssh root@115.190.239.146 "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
```

**输入密码**: `GGsuperman1211`

### 第 2 步:验证连接 ✓

```bash
ssh root@115.190.239.146 "echo 'SSH OK'"
```

应该显示 "SSH OK" 且无需密码。

### 第 3 步:执行部署 🚀

```bash
cd /d/tools/lurus-switch
bash scripts/final-deploy.sh
```

## 部署内容

脚本将自动部署以下服务:

### 基础设施层
- **PostgreSQL** (5432) - 主数据库
- **Redis** (6379) - 缓存和队列
- **NATS** (4222) - 消息总线
- **ClickHouse** (8123/9000) - 日志分析数据库
- **Consul** (8500) - 服务发现

### 可观测性层
- **Prometheus** (9090) - 指标收集
- **Grafana** (3000) - 可视化面板
- **Jaeger** (16686) - 分布式追踪
- **Alertmanager** (9093) - 告警管理

## 部署后访问

| 服务 | 地址 | 凭据 |
|------|------|------|
| Grafana | http://115.190.239.146:3000 | admin / admin |
| Prometheus | http://115.190.239.146:9090 | 无需认证 |
| Jaeger UI | http://115.190.239.146:16686 | 无需认证 |
| Consul UI | http://115.190.239.146:8500 | 无需认证 |

## 部署脚本说明

我们提供了多个部署脚本,您可以选择最适合的:

| 脚本 | 说明 | 推荐指数 |
|------|------|---------|
| `scripts/final-deploy.sh` | 一键部署脚本 (Bash) | ⭐⭐⭐⭐⭐ |
| `Deploy-Quick.ps1` | 简化部署脚本 (PowerShell) | ⭐⭐⭐⭐ |
| `scripts/quick-deploy.sh` | 交互式部署脚本 | ⭐⭐⭐ |
| `QUICK-START.md` | 手动分步指南 | ⭐⭐⭐⭐ |
| `DEPLOYMENT-STEPS.md` | 详细手动指南 | ⭐⭐⭐⭐⭐ |

## 常用命令

### 查看服务状态

```bash
ssh root@115.190.239.146 "docker ps"
```

### 查看日志

```bash
# PostgreSQL
ssh root@115.190.239.146 "docker logs -f lurus-postgres"

# Redis
ssh root@115.190.239.146 "docker logs -f lurus-redis"

# NATS
ssh root@115.190.239.146 "docker logs -f lurus-nats"

# Grafana
ssh root@115.190.239.146 "docker logs -f lurus-grafana"
```

### 重启服务

```bash
# 重启所有服务
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml restart"

# 重启单个服务
ssh root@115.190.239.146 "docker restart lurus-postgres"
```

### 停止所有服务

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml down"
```

### 完全重新部署

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml down -v && docker compose -f docker-compose.dev.yaml up -d"
```

## 下一步:部署微服务

基础设施部署完成后,还需要部署微服务:

1. **Gateway Service** (18100) - API 网关
2. **Provider Service** (18101) - 供应商管理
3. **Log Service** (18102) - 日志服务
4. **Billing Service** (18103) - 计费服务

### 部署微服务步骤

```bash
# 1. 复制源代码到服务器
scp -r gateway-service provider-service log-service billing-service lurus-common root@115.190.239.146:/opt/lurus/

# 2. 在服务器上构建
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml build"

# 3. 启动微服务
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d gateway-service provider-service log-service billing-service"

# 4. 检查状态
ssh root@115.190.239.146 "docker ps | grep 'gateway\|provider\|log\|billing'"
```

## 监控和维护

### 查看资源使用

```bash
ssh root@115.190.239.146 "docker stats --no-stream"
```

### 清理磁盘空间

```bash
ssh root@115.190.239.146 "docker system prune -af"
```

### 备份数据库

```bash
# PostgreSQL 备份
ssh root@115.190.239.146 "docker exec lurus-postgres pg_dump -U lurus lurus > /opt/lurus/backup/lurus_$(date +%Y%m%d).sql"
```

## 故障排查

### 问题 1: 容器无法启动

```bash
# 查看容器日志
ssh root@115.190.239.146 "docker logs <container-name>"

# 检查容器配置
ssh root@115.190.239.146 "docker inspect <container-name>"
```

### 问题 2: 端口被占用

```bash
# 检查端口占用
ssh root@115.190.239.146 "netstat -tlnp | grep <port>"
```

### 问题 3: Docker 镜像拉取失败

```bash
# 配置国内镜像源
ssh root@115.190.239.146 "cat > /etc/docker/daemon.json << 'EOF'
{
  \"registry-mirrors\": [
    \"https://docker.mirrors.ustc.edu.cn\",
    \"https://hub-mirror.c.163.com\"
  ]
}
EOF
systemctl restart docker
"
```

## 相关文档

- 📖 [快速启动指南](QUICK-START.md) - 最简单的部署方式
- 📖 [详细部署步骤](DEPLOYMENT-STEPS.md) - 完整的手动部署指南
- 📖 [多节点部署指南](doc/multi-node-deployment-guide.md) - 生产环境部署架构
- 📖 [Docker Compose 配置](docker-compose.dev.yaml) - 服务配置详情

## 联系和支持

如果遇到问题,请检查:

1. SSH 连接是否正常: `ssh root@115.190.239.146 "echo OK"`
2. Docker 服务是否运行: `ssh root@115.190.239.146 "docker ps"`
3. 容器日志: `ssh root@115.190.239.146 "docker logs <container-name>"`

---

**部署时间预计**: 5-10 分钟 (取决于网络速度)
**系统要求**: Docker 20+, Docker Compose V2
**推荐配置**: 8GB+ RAM, 50GB+ 磁盘空间
