# Lurus Switch Windows Native Deployment
# Windows 原生部署方案

> 适用于无法运行 Linux 容器的 Windows Server 环境（如阿里云 VM 无嵌套虚拟化）

## 架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Internet                                      │
└───────────────────────────┬─────────────────────────────────────────┘
                            │ :80/:443
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Caddy (Edge Proxy - Windows)                       │
│  ├─ api.lurus.cn    → new-api:3000                                  │
│  ├─ ai.lurus.cn     → gateway-service:18100                         │
│  ├─ portal.lurus.cn → static files                                  │
│  └─ grafana.lurus.cn → grafana:3000                                 │
└───────────────────────────┬─────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌───────────────┐  ┌────────────────┐  ┌────────────────┐
│   new-api     │  │ gateway-service│  │ sync-service   │
│   (Go/Win)    │  │   (Go/Win)     │  │   (Go/Win)     │
│   :3000       │  │    :18100      │  │   :8081        │
└───────────────┘  └────────────────┘  └────────────────┘
                            │
┌───────────────────────────┴───────────────────────────┐
│                    Infrastructure                      │
│  PostgreSQL (Win) | Redis (Win) | NATS (Win)          │
└───────────────────────────────────────────────────────┘
```

## 组件清单

| 组件 | Windows 版本 | 下载地址 |
|------|-------------|---------|
| Caddy | caddy_windows_amd64.exe | https://caddyserver.com/download |
| PostgreSQL | 16.x Windows installer | https://www.postgresql.org/download/windows/ |
| Redis | Memurai (Windows Redis) | https://www.memurai.com/ |
| NATS | nats-server-windows-amd64 | https://nats.io/download/ |
| Grafana | grafana-windows-amd64 | https://grafana.com/grafana/download |
| new-api | 自行编译 (Go) | `GOOS=windows go build` |
| gateway-service | 自行编译 (Go) | `GOOS=windows go build` |
| sync-service | 自行编译 (Go) | `GOOS=windows go build` |

## 安装步骤

### 1. 安装 PostgreSQL

```powershell
# 下载安装包
Invoke-WebRequest -Uri "https://get.enterprisedb.com/postgresql/postgresql-16.4-1-windows-x64.exe" -OutFile "D:\install\postgresql-16.exe"

# 运行安装程序（或静默安装）
D:\install\postgresql-16.exe --mode unattended --superpassword YOUR_PASSWORD --servicename postgresql --datadir D:\data\postgresql
```

### 2. 安装 Redis (Memurai)

```powershell
# 下载 Memurai（Windows 兼容 Redis）
Invoke-WebRequest -Uri "https://www.memurai.com/get-memurai" -OutFile "D:\install\memurai.msi"

# 安装
msiexec /i D:\install\memurai.msi /qn
```

### 3. 安装 NATS Server

```powershell
# 下载 NATS
Invoke-WebRequest -Uri "https://github.com/nats-io/nats-server/releases/download/v2.10.24/nats-server-v2.10.24-windows-amd64.zip" -OutFile "D:\install\nats-server.zip"

# 解压
Expand-Archive -Path "D:\install\nats-server.zip" -DestinationPath "D:\services\nats"

# 创建配置
@"
port: 4222
http_port: 8222
jetstream {
    store_dir: "D:/data/nats/jetstream"
}
"@ | Set-Content "D:\services\nats\nats-server.conf"

# 注册为 Windows 服务
sc.exe create nats-server binPath= "D:\services\nats\nats-server.exe -c D:\services\nats\nats-server.conf" start= auto
```

### 4. 安装 Caddy

```powershell
# 下载 Caddy
Invoke-WebRequest -Uri "https://caddyserver.com/api/download?os=windows&arch=amd64" -OutFile "D:\services\caddy\caddy.exe"

# 复制 Caddyfile
Copy-Item "D:\tools\lurus-switch\deploy\caddy\Caddyfile" "D:\services\caddy\Caddyfile"

# 注册为 Windows 服务
sc.exe create caddy binPath= "D:\services\caddy\caddy.exe run --config D:\services\caddy\Caddyfile" start= auto
```

### 5. 编译 Go 服务

```powershell
# new-api
cd D:\tools\lurus-switch\new-api
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -o D:\services\new-api\new-api.exe .

# gateway-service (当创建后)
cd D:\tools\lurus-switch\gateway-service
go build -o D:\services\gateway-service\gateway-service.exe ./cmd/gateway

# sync-service
cd D:\tools\lurus-switch\sync-service
go build -o D:\services\sync-service\sync-service.exe .
```

### 6. 注册 Windows 服务

使用 NSSM (Non-Sucking Service Manager) 注册服务:

```powershell
# 下载 NSSM
Invoke-WebRequest -Uri "https://nssm.cc/release/nssm-2.24.zip" -OutFile "D:\install\nssm.zip"
Expand-Archive -Path "D:\install\nssm.zip" -DestinationPath "D:\tools"

# 注册 new-api 服务
D:\tools\nssm-2.24\win64\nssm.exe install new-api D:\services\new-api\new-api.exe
D:\tools\nssm-2.24\win64\nssm.exe set new-api AppDirectory D:\services\new-api
D:\tools\nssm-2.24\win64\nssm.exe set new-api AppEnvironmentExtra SQL_DSN=postgres://lurus:PASSWORD@localhost:5432/new_api

# 类似地注册其他服务...
```

## 服务启动顺序

1. PostgreSQL
2. Redis (Memurai)
3. NATS Server
4. new-api
5. gateway-service
6. sync-service
7. Caddy

## 环境变量配置

创建 `D:\services\env.ps1`:

```powershell
# Database
$env:SQL_DSN = "postgres://lurus:PASSWORD@localhost:5432/new_api"
$env:REDIS_CONN_STRING = "localhost:6379"

# NATS
$env:NATS_URL = "nats://localhost:4222"

# Session
$env:SESSION_SECRET = "your-session-secret-here"

# Ports
$env:PORT = "3000"
```

## 与 Docker 方案的差异

| 方面 | Docker 方案 | Windows 原生方案 |
|------|------------|-----------------|
| 隔离性 | 容器隔离 | 进程隔离 |
| 部署 | docker-compose up | 手动/脚本 |
| 升级 | 重新拉取镜像 | 重新编译/下载 |
| 监控 | Docker stats | Windows 性能监视器 |
| 日志 | Docker logs | Windows 事件日志/文件 |
| 网络 | Docker network | Windows 防火墙 |

## 注意事项

1. **ClickHouse**: 官方不提供 Windows 版本，需要使用 WSL 或远程服务器
   - 替代方案：使用 TimescaleDB (PostgreSQL 扩展) 或云托管 ClickHouse

2. **Consul**: 有 Windows 版本，但如果只是本机部署可以跳过服务发现

3. **防火墙**: 需要开放相应端口
   ```powershell
   New-NetFirewallRule -DisplayName "Lurus Services" -Direction Inbound -LocalPort 80,443,3000,8081,18100 -Protocol TCP -Action Allow
   ```

4. **SSL 证书**: Caddy 自动管理 Let's Encrypt 证书，确保域名 DNS 已配置
