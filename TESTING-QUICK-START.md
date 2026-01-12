# CodeSwitch 快速测试指南

## 测试目标

本指南帮助你快速验证 CodeSwitch 的核心功能:
- ✅ 基础设施服务正常运行
- ✅ 代理转发功能工作正常
- ✅ 数据库日志记录正常
- ✅ 现有单元测试通过

## 前置要求

- Docker 已安装并运行
- Go 1.24+ 已安装
- NATS CLI 已安装 (可选,用于 NATS 测试)
- PostgreSQL CLI 已安装 (可选,用于数据库验证)

## 快速测试步骤

### 第 1 步: 启动基础设施 (3 分钟)

```powershell
# 进入 docker 目录
cd D:\tools\lurus-switch\codeswitch\deploy\docker

# 查看配置 (确认密码等配置)
cat .env

# 启动核心服务
docker-compose up -d postgres redis nats

# 等待服务就绪
Start-Sleep -Seconds 30

# 运行健康检查
D:\tools\lurus-switch\scripts\health-check.ps1
```

**预期输出:**
```
PostgreSQL       :5432    [ OK ]
Redis            :6379    [ OK ]
NATS             :4222    [ OK ]
```

### 第 2 步: 初始化 NATS (1 分钟)

```powershell
# 运行 NATS 初始化脚本
D:\tools\lurus-switch\scripts\init-nats-streams.ps1
```

**预期输出:**
```
Creating stream: CHAT_MESSAGES...
  Created successfully
...
Summary:
  Created: 6
```

### 第 3 步: 运行现有单元测试 (2 分钟)

```powershell
# 进入 codeswitch 目录
cd D:\tools\lurus-switch\codeswitch

# 运行所有单元测试
go test ./services/... -v

# 或只运行特定测试
go test ./services/providerservice_test.go -v
go test ./services/providerrelay_test.go -v
```

**预期输出:**
```
=== RUN   TestProviderService
--- PASS: TestProviderService (0.00s)
...
PASS
ok      codeswitch/services     1.234s
```

### 第 4 步: 测试代理功能 (手动,3 分钟)

#### 4.1 启动 CodeSwitch Desktop App

```powershell
# 开发模式启动
wails3 task dev

# 或构建并运行
wails3 task build
.\build\bin\CodeSwitch.exe
```

#### 4.2 配置测试 Provider

在 CodeSwitch GUI 中:
1. 点击 "Add Provider"
2. 填写测试配置:
   - Name: `Test-Provider`
   - API URL: `https://httpbin.org` (用于测试,返回请求信息)
   - API Key: `test-key`
   - Supported Models: `test-model`
3. 保存

#### 4.3 发送测试请求

```powershell
# 测试 Claude Code 代理端点
curl http://localhost:18100/v1/messages `
  -H "Content-Type: application/json" `
  -d '{
    "model": "test-model",
    "messages": [{"role": "user", "content": "Hello Test"}]
  }'
```

**预期:**
- 请求被转发 (即使失败也说明代理工作)
- GUI 日志显示请求记录

#### 4.4 查看日志

在 CodeSwitch GUI 中:
1. 导航到 "Logs" 页面
2. 应该看到刚才的测试请求
3. 验证字段: Model, Provider, Tokens, Cost

### 第 5 步: 验证数据库日志 (2 分钟)

```powershell
# 连接到 PostgreSQL (如果已安装 psql)
$env:PGPASSWORD = "CodeSwitch_Test_2026!"
psql -h localhost -U codeswitch -d codeswitch

# 查询最近的请求日志
SELECT id, platform, model, provider, http_code, created_at
FROM request_log
ORDER BY created_at DESC
LIMIT 10;

# 退出
\q
```

**预期:**
看到刚才的测试请求记录,包含正确的 model, provider 等信息。

---

## 快速验证清单

完成上述步骤后,核对以下清单:

- [ ] PostgreSQL 运行正常 (端口 5432)
- [ ] Redis 运行正常 (端口 6379)
- [ ] NATS 运行正常 (端口 4222, Monitor 8222)
- [ ] NATS JetStream 创建了 6 个流
- [ ] 单元测试全部通过 (`go test ./services/...`)
- [ ] CodeSwitch Desktop 启动成功 (端口 18100)
- [ ] 代理请求可以发送 (即使失败也说明代理工作)
- [ ] 请求日志被正确记录到数据库

---

## 常见问题排查

### 问题 1: Docker 容器启动失败

```powershell
# 查看日志
docker logs codeswitch-postgres
docker logs codeswitch-redis
docker logs codeswitch-nats

# 检查端口占用
netstat -ano | findstr :5432
netstat -ano | findstr :6379
netstat -ano | findstr :4222

# 重启 Docker
# (通过任务栏 Docker Desktop 图标)
```

### 问题 2: 单元测试失败

```powershell
# 查看详细错误
go test ./services/providerservice_test.go -v -run TestSpecificTest

# 清理并重新测试
go clean -testcache
go test ./services/... -v
```

### 问题 3: CodeSwitch 编译失败

```powershell
# 检查 Wails 版本
wails3 version

# 清理缓存
wails3 clean

# 重新生成绑定
wails3 generate bindings

# 重新构建
wails3 task build
```

### 问题 4: 代理请求无响应

**检查步骤:**
1. 确认 CodeSwitch 监听 :18100
   ```powershell
   netstat -ano | findstr :18100
   ```

2. 查看 CodeSwitch 控制台日志

3. 确认 Provider 配置正确:
   - API URL 有效
   - Supported Models 包含请求的模型

---

## 性能基准测试 (可选)

### 并发请求测试

```powershell
# 使用 Apache Bench (如已安装)
# 100 请求, 10 并发
ab -n 100 -c 10 `
  -p test-request.json `
  -T application/json `
  http://localhost:18100/v1/messages
```

**创建 test-request.json:**
```json
{
  "model": "test-model",
  "messages": [{"role": "user", "content": "Benchmark test"}]
}
```

### 日志写入性能

```sql
-- 连接到数据库后执行
-- 查看最近 1 小时的请求统计
SELECT
  COUNT(*) as total_requests,
  AVG(duration_sec) as avg_duration,
  MAX(duration_sec) as max_duration,
  SUM(input_tokens) as total_input_tokens,
  SUM(output_tokens) as total_output_tokens
FROM request_log
WHERE created_at > NOW() - INTERVAL '1 hour';
```

---

## 下一步: 完整测试

完成快速测试后,如需进行完整测试,请参考:
- **D:\tools\lurus-switch\codeswitch\CLAUDE.md** - 完整测试计划
- **D:\tools\lurus-switch\scripts\** - 自动化测试脚本

---

## 测试报告模板

完成测试后,记录结果:

```
# CodeSwitch 快速测试报告

**测试日期:** YYYY-MM-DD
**测试人员:** Your Name
**环境:** Windows Server 2019

## 测试结果

### 基础设施
- PostgreSQL: [ ] Pass [ ] Fail
- Redis: [ ] Pass [ ] Fail
- NATS: [ ] Pass [ ] Fail

### 单元测试
- ProviderService: [ ] Pass [ ] Fail
- ProviderRelay: [ ] Pass [ ] Fail
- 覆盖率: XX%

### 功能测试
- 代理转发: [ ] Pass [ ] Fail
- 日志记录: [ ] Pass [ ] Fail
- 数据库写入: [ ] Pass [ ] Fail

## 发现的问题
1. [问题描述]
2. [问题描述]

## 建议
1. [建议内容]
2. [建议内容]
```

---

**创建时间:** 2026-01-11
**维护者:** CodeSwitch Team
**更新频率:** 每次重大更新后
