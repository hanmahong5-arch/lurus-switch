# Lurus Switch 部署到 115.190.239.146 服务器

## 第一步:配置 SSH 密钥认证 (只需执行一次)

在您的本地终端(Git Bash或PowerShell)执行以下命令:

```bash
cat ~/.ssh/id_rsa.pub | ssh root@115.190.239.146 "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo 'SSH key configured successfully!'"
```

**当提示输入密码时,输入**: `GGsuperman1211`

执行成功后,您会看到 "SSH key configured successfully!" 消息。

## 第二步:验证 SSH 连接

```bash
ssh root@115.190.239.146 "echo 'Connection OK' && uname -a"
```

应该能够免密登录并显示系统信息。

## 第三步:执行部署脚本

### 方式 A: 使用 Bash 脚本 (推荐)

```bash
cd /d/tools/lurus-switch
bash scripts/deploy-to-server.sh
```

### 方式 B: 使用 PowerShell 脚本

```powershell
cd D:\tools\lurus-switch
pwsh scripts/Deploy-ToServer.ps1 -DeployAll
```

### 方式 C: 手动分步部署

#### 3.1 检查服务器环境

```bash
ssh root@115.190.239.146 "
    echo '=== System Info ===' &&
    uname -a &&
    echo '' &&
    echo '=== Docker Version ===' &&
    docker --version &&
    docker compose version &&
    echo '' &&
    echo '=== Resources ===' &&
    free -h &&
    df -h /
"
```

#### 3.2 创建目录并复制文件

```bash
cd /d/tools/lurus-switch

# 创建目录
ssh root@115.190.239.146 "mkdir -p /opt/lurus/{deploy,data,logs,backup}"

# 复制配置文件
scp docker-compose.dev.yaml root@115.190.239.146:/opt/lurus/
scp -r deploy root@115.190.239.146:/opt/lurus/
scp -r prometheus.yml root@115.190.239.146:/opt/lurus/
```

#### 3.3 创建环境变量文件

```bash
ssh root@115.190.239.146 "cat > /opt/lurus/.env << 'EOF'
# PostgreSQL
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_secure_2026
POSTGRES_DB=lurus

# ClickHouse
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_secure_2026
CLICKHOUSE_DB=lurus_logs

# Grafana
GF_SECURITY_ADMIN_PASSWORD=admin

# Redis (optional)
REDIS_PASSWORD=

# Session Secret
SESSION_SECRET=$(openssl rand -hex 32)
EOF
"
```

#### 3.4 拉取 Docker 镜像

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml pull"
```

#### 3.5 启动基础设施服务

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"
```

等待 20 秒让服务初始化:

```bash
sleep 20
```

#### 3.6 检查容器状态

```bash
ssh root@115.190.239.146 "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
```

#### 3.7 初始化数据库

```bash
ssh root@115.190.239.146 "docker exec lurus-postgres psql -U lurus -c '\l'"
```

#### 3.8 启动可观测性服务

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"
```

#### 3.9 查看所有服务状态

```bash
ssh root@115.190.239.146 "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
```

## 第四步:访问服务

假设服务器 IP 为 115.190.239.146:

- **Grafana**: http://115.190.239.146:3000 (用户名: admin, 密码: admin)
- **Prometheus**: http://115.190.239.146:9090
- **Jaeger UI**: http://115.190.239.146:16686
- **Consul UI**: http://115.190.239.146:8500
- **PostgreSQL**: 115.190.239.146:5432
- **Redis**: 115.190.239.146:6379
- **NATS**: 115.190.239.146:4222

## 第五步:部署微服务 (需要先构建镜像)

微服务部署需要先构建 Docker 镜像。有两种方式:

### 方式 A: 在服务器上构建

```bash
# 复制代码到服务器
scp -r gateway-service provider-service log-service billing-service lurus-common root@115.190.239.146:/opt/lurus/

# 在服务器上构建
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml build"

# 启动服务
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d gateway-service provider-service log-service billing-service"
```

### 方式 B: 本地构建并推送到 Docker Registry

```bash
# 这需要先配置 Docker Registry (GitHub Container Registry 或 Docker Hub)
# 详见后续文档
```

## 故障排查

### 查看日志

```bash
# 查看特定容器日志
ssh root@115.190.239.146 "docker logs -f lurus-postgres"
ssh root@115.190.239.146 "docker logs -f lurus-nats"

# 查看所有容器日志摘要
ssh root@115.190.239.146 "docker ps --format '{{.Names}}' | xargs -I {} sh -c 'echo === {} === && docker logs --tail 10 {}'"
```

### 重启服务

```bash
# 重启单个服务
ssh root@115.190.239.146 "docker restart lurus-postgres"

# 重启所有服务
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml restart"
```

### 停止所有服务

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml down"
```

### 清理并重新部署

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml down -v && docker compose -f docker-compose.dev.yaml up -d"
```

## 后续步骤

1. **配置域名**: 如果有域名,配置 DNS 指向服务器 IP
2. **配置 HTTPS**: 使用 Caddy 或 Nginx 配置 SSL 证书
3. **配置防火墙**: 开放必要的端口
4. **配置备份**: 设置定时备份脚本
5. **配置监控告警**: 在 Grafana 中配置告警规则
