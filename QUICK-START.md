# Lurus Switch 快速部署指南

服务器IP: **115.190.239.146**
用户名: **root**
密码: **GGsuperman1211**

## 方式一:最简单的三步部署 (推荐)

### 第 1 步:配置 SSH 密钥 (只需一次)

在 **Git Bash** 或 **PowerShell** 中执行:

```bash
cat ~/.ssh/id_rsa.pub | ssh root@115.190.239.146 "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
```

输入密码: `GGsuperman1211`

### 第 2 步:验证 SSH 连接

```bash
ssh root@115.190.239.146 "echo 'Connection OK'"
```

应该显示 "Connection OK" 而不需要输入密码。

### 第 3 步:运行部署脚本

```bash
cd D:\tools\lurus-switch
powershell.exe -ExecutionPolicy Bypass -File Deploy-Quick.ps1
```

或者使用 Bash:

```bash
cd /d/tools/lurus-switch
bash scripts/deploy-to-server.sh
```

## 方式二:手动分步部署

### 1. 测试连接

```bash
ssh root@115.190.239.146 "uname -a && docker --version"
```

### 2. 创建目录

```bash
ssh root@115.190.239.146 "mkdir -p /opt/lurus/{deploy,data,logs,backup}"
```

### 3. 复制文件

```bash
cd D:\tools\lurus-switch
scp docker-compose.dev.yaml root@115.190.239.146:/opt/lurus/
scp -r deploy root@115.190.239.146:/opt/lurus/
```

### 4. 启动基础设施

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"
```

### 5. 等待服务启动 (30秒)

```bash
sleep 30
```

### 6. 启动监控服务

```bash
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"
```

### 7. 检查状态

```bash
ssh root@115.190.239.146 "docker ps"
```

## 服务访问地址

部署完成后,您可以访问:

- **Grafana**: http://115.190.239.146:3000
  - 用户名: `admin`
  - 密码: `admin`

- **Prometheus**: http://115.190.239.146:9090

- **Jaeger UI**: http://115.190.239.146:16686

- **Consul UI**: http://115.190.239.146:8500

- **PostgreSQL**: `115.190.239.146:5432`
  - 用户名: `lurus`
  - 密码: (在 .env 文件中)

- **Redis**: `115.190.239.146:6379`

- **NATS**: `115.190.239.146:4222`

- **ClickHouse HTTP**: `115.190.239.146:8123`

## 常用命令

### 查看日志

```bash
# 查看所有容器
ssh root@115.190.239.146 "docker ps"

# 查看特定容器日志
ssh root@115.190.239.146 "docker logs -f lurus-postgres"
ssh root@115.190.239.146 "docker logs -f lurus-redis"
ssh root@115.190.239.146 "docker logs -f lurus-nats"
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

### 查看资源使用

```bash
ssh root@115.190.239.146 "docker stats --no-stream"
```

## 下一步:部署微服务

基础设施部署完成后,需要构建并部署微服务。

### 方式 A:在服务器上构建

```bash
# 1. 复制源代码
scp -r gateway-service provider-service log-service billing-service lurus-common root@115.190.239.146:/opt/lurus/

# 2. 构建服务
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml build"

# 3. 启动微服务
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d gateway-service provider-service log-service billing-service"
```

### 方式 B:使用预构建镜像 (需要 Docker Registry)

```bash
# 本地构建并推送
docker compose -f docker-compose.dev.yaml build
docker compose -f docker-compose.dev.yaml push

# 服务器拉取并启动
ssh root@115.190.239.146 "cd /opt/lurus && docker compose -f docker-compose.dev.yaml pull && docker compose -f docker-compose.dev.yaml up -d"
```

## 故障排查

### 问题 1: Docker 镜像拉取失败

```bash
# 切换到国内镜像源
ssh root@115.190.239.146 "cat > /etc/docker/daemon.json << 'EOF'
{
  \"registry-mirrors\": [
    \"https://docker.mirrors.ustc.edu.cn\",
    \"https://hub-mirror.c.163.com\"
  ]
}
EOF
"
ssh root@115.190.239.146 "systemctl restart docker"
```

### 问题 2: 容器无法启动

```bash
# 查看容器日志
ssh root@115.190.239.146 "docker logs <container-name>"

# 查看容器详情
ssh root@115.190.239.146 "docker inspect <container-name>"
```

### 问题 3: 端口被占用

```bash
# 查看端口占用
ssh root@115.190.239.146 "netstat -tlnp | grep <port>"

# 或使用 ss
ssh root@115.190.239.146 "ss -tlnp | grep <port>"
```

### 问题 4: 磁盘空间不足

```bash
# 清理未使用的镜像和容器
ssh root@115.190.239.146 "docker system prune -af"
```

## 备份和恢复

### 备份数据库

```bash
# PostgreSQL
ssh root@115.190.239.146 "docker exec lurus-postgres pg_dump -U lurus lurus > /opt/lurus/backup/lurus_$(date +%Y%m%d).sql"

# ClickHouse
ssh root@115.190.239.146 "docker exec lurus-clickhouse clickhouse-client --query='BACKUP DATABASE lurus_logs TO Disk('\"backups\"', '\"backup_$(date +%Y%m%d).zip\"')'"
```

### 恢复数据库

```bash
# PostgreSQL
ssh root@115.190.239.146 "docker exec -i lurus-postgres psql -U lurus lurus < /opt/lurus/backup/lurus_20260112.sql"
```

## 参考文档

- 完整部署指南: `doc/multi-node-deployment-guide.md`
- 详细部署步骤: `DEPLOYMENT-STEPS.md`
- Docker Compose配置: `docker-compose.dev.yaml`
