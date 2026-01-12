# CodeSwitch 测试执行指南

## 概述

本指南说明如何运行 CodeSwitch 项目的全部测试套件,包括:
- **单元测试**: Provider 服务、日志服务、代理转发
- **集成测试**: NATS、NEW-API、计费系统
- **E2E 测试**: 完整用户流程 (需单独配置 Playwright)

---

## 前置准备

### 1. 环境要求

```powershell
# 检查 Go 版本 (需要 1.24+)
go version

# 检查 Docker 运行状态
docker ps

# 检查测试依赖
cd D:\tools\lurus-switch\codeswitch
go mod download
go mod tidy
```

### 2. 启动测试基础设施

**必需服务** (单元测试和集成测试):
```powershell
# 进入 docker 目录
cd D:\tools\lurus-switch\codeswitch\deploy\docker

# 启动核心服务
docker-compose up -d postgres redis nats

# 等待服务就绪
Start-Sleep -Seconds 30

# 验证健康状态
D:\tools\lurus-switch\scripts\health-check.ps1
```

**可选服务** (完整集成测试):
```powershell
# 启动完整栈 (Casdoor, Lago, NEW-API)
docker-compose up -d

# 初始化 NATS JetStream
D:\tools\lurus-switch\scripts\init-nats-streams.ps1
```

---

## 运行测试

### 方式 1: 运行所有单元测试 (推荐)

```powershell
cd D:\tools\lurus-switch\codeswitch

# 运行所有 services 目录下的测试
go test ./services/... -v

# 查看测试覆盖率
go test ./services/... -cover

# 生成详细覆盖率报告
go test ./services/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**预期输出:**
```
=== RUN   TestProviderRelay_ClaudeCodeProxy_Success
--- PASS: TestProviderRelay_ClaudeCodeProxy_Success (0.05s)
=== RUN   TestProviderRelay_LevelFallback
--- PASS: TestProviderRelay_LevelFallback (0.03s)
...
PASS
coverage: 68.5% of statements
ok      codeswitch/services     2.456s
```

### 方式 2: 运行特定测试文件

```powershell
# Claude Code 代理测试
go test ./services/providerrelay_claude_test.go -v

# Provider 降级测试
go test ./services/providerrelay_fallback_test.go -v

# 流式响应测试
go test ./services/providerrelay_stream_test.go -v

# 日志服务测试
go test ./services/logservice_test.go -v

# 现有的测试 (模型映射等)
go test ./services/providerrelay_test.go -v
go test ./services/providerservice_test.go -v
```

### 方式 3: 运行特定测试用例

```powershell
# 运行单个测试函数
go test ./services/... -v -run TestProviderRelay_ClaudeCodeProxy_Success

# 运行匹配模式的测试
go test ./services/... -v -run ".*Fallback.*"

# 运行带超时的测试
go test ./services/... -v -timeout 30s
```

### 方式 4: 并发测试 (压力测试)

```powershell
# 使用多个 CPU 核心并行测试
go test ./services/... -v -parallel 4

# 重复运行 10 次检测竞态条件
go test ./services/... -v -count=10

# 启用竞态检测 (Race Detector)
go test ./services/... -v -race
```

---

## 测试分类

### P0 核心测试 (必须全部通过)

```powershell
# 代理转发核心功能
go test ./services/... -v -run "TestProviderRelay_ClaudeCodeProxy.*"

# Provider 选择和降级
go test ./services/... -v -run "TestProviderRelay_.*Fallback.*"

# 流式响应处理
go test ./services/... -v -run "TestProviderRelay_Streaming.*"
```

### P1 重要测试 (通过率 >= 90%)

```powershell
# 日志批量写入
go test ./services/... -v -run "TestLogService_BatchWrite"

# 价格计算
go test ./services/... -v -run "TestLogService_Price.*"

# 并发读写
go test ./services/... -v -run "TestLogService_ConcurrentReadWrite"
```

### P2 辅助测试 (可选)

```powershell
# 配置管理
go test ./services/... -v -run "TestProviderService.*"

# 模型映射
go test ./services/... -v -run "TestReplaceModelInRequestBody"
```

---

## 测试结果解读

### 成功输出示例

```
=== RUN   TestProviderRelay_ClaudeCodeProxy_Success
=== PAUSE TestProviderRelay_ClaudeCodeProxy_Success
=== CONT  TestProviderRelay_ClaudeCodeProxy_Success
--- PASS: TestProviderRelay_ClaudeCodeProxy_Success (0.21s)
    providerrelay_claude_test.go:89: Price breakdown: Input=$0.0030, Output=$0.0150, Total=$0.0180
PASS
ok      codeswitch/services     0.456s
```

### 失败输出示例

```
=== RUN   TestProviderRelay_LevelFallback
--- FAIL: TestProviderRelay_LevelFallback (0.11s)
    providerrelay_fallback_test.go:65:
                Error Trace:    providerrelay_fallback_test.go:65
                Error:          Not equal:
                                expected: 1
                                actual  : 0
                Test:           TestProviderRelay_LevelFallback
                Messages:       Level 1 should be tried first
FAIL
FAIL    codeswitch/services     0.345s
```

**常见失败原因:**
1. **数据库未启动**: `dial tcp [::1]:5432: connect: connection refused`
   - 解决: 启动 PostgreSQL (`docker-compose up -d postgres`)

2. **测试超时**: `panic: test timed out after 2m0s`
   - 解决: 增加超时 `go test -timeout 5m ...`

3. **竞态条件**: `WARNING: DATA RACE`
   - 解决: 检查并发访问的代码,使用互斥锁保护共享变量

4. **日志队列满**: `Log queue full for trace-XX`
   - 解决: 这是预期行为,测试验证溢出处理逻辑

---

## 集成测试

### NATS 消息同步测试

```powershell
# 启动 NATS (如果未启动)
docker-compose up -d nats

# 初始化 Streams
D:\tools\lurus-switch\scripts\init-nats-streams.ps1

# 运行 NATS 集成测试 (需要 Sync Service 运行)
cd D:\tools\lurus-switch\codeswitch\sync-service
go run cmd/main.go &

# 验证 NATS 消息发布
nats sub "llm.request.*" -s nats://localhost:4222
# (在另一个终端发送测试请求)
curl http://localhost:18100/v1/messages -d '{"model":"claude-sonnet-4","messages":[{"role":"user","content":"test"}]}'
```

### NEW-API 模式测试

```powershell
# 启动 NEW-API
cd D:\tools\lurus-switch\new-api
.\new-api.exe &

# 配置 CodeSwitch 启用 NEW-API 模式
# 编辑 ~/.code-switch/app.json:
# {
#   "new_api_enabled": true,
#   "new_api_url": "http://localhost:3000",
#   "new_api_token": "sk-your-token"
# }

# 启动 CodeSwitch
cd D:\tools\lurus-switch\codeswitch
wails3 task dev

# 测试请求通过 NEW-API 转发
curl http://localhost:18100/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Test NEW-API"}]}'

# 验证: 查看 NEW-API 日志,应该显示请求记录
```

### Lago 计费测试

```powershell
# 启动 Lago 服务
docker-compose up -d lago-api lago-front

# 访问 Lago UI: http://localhost:8080
# 创建 Billable Metrics 和订阅计划

# 配置 CodeSwitch billing
# 编辑 ~/.code-switch/billing-config.json

# 运行计费测试
cd D:\tools\lurus-switch\codeswitch
go test ./services/billing_integration_test.go -v
```

---

## E2E 测试 (Playwright)

### 安装 Playwright (首次)

```powershell
cd D:\tools\lurus-switch\codeswitch\frontend

# 安装 Playwright
npm install -D @playwright/test
npx playwright install chromium

# 配置 Playwright (参考计划文档中的配置)
```

### 运行 E2E 测试

```powershell
# 确保 CodeSwitch 正在运行
cd D:\tools\lurus-switch\codeswitch
wails3 task dev &

# 运行 E2E 测试
cd frontend
npm run test:e2e

# 交互式 UI 模式
npm run test:e2e:ui

# 调试模式
npm run test:e2e:debug

# 查看报告
npm run test:e2e:report
```

**E2E 测试用例:**
- 首次启动和配置流程
- 供应商添加/编辑/删除
- 日志查看和搜索
- 统计数据刷新
- NEW-API 网关配置

---

## 性能基准测试

### 并发请求测试

```powershell
# 使用 Apache Bench (如已安装)
# 100 请求, 10 并发
ab -n 100 -c 10 \
  -p test-request.json \
  -T application/json \
  http://localhost:18100/v1/messages
```

**test-request.json:**
```json
{
  "model": "claude-sonnet-4",
  "messages": [{"role": "user", "content": "Benchmark test"}],
  "max_tokens": 100
}
```

### 日志写入性能测试

```sql
-- 连接到 PostgreSQL
$env:PGPASSWORD = "CodeSwitch_Test_2026!"
psql -h localhost -U codeswitch -d codeswitch

-- 查看最近 1 小时的请求统计
SELECT
  COUNT(*) as total_requests,
  AVG(duration_sec) as avg_duration,
  MAX(duration_sec) as max_duration,
  SUM(input_tokens) as total_input_tokens,
  SUM(output_tokens) as total_output_tokens,
  SUM(total_cost) as total_cost
FROM request_log
WHERE created_at > NOW() - INTERVAL '1 hour';

-- 查看写入性能 (每秒请求数)
SELECT
  DATE_TRUNC('second', created_at) as second,
  COUNT(*) as requests_per_second
FROM request_log
WHERE created_at > NOW() - INTERVAL '5 minutes'
GROUP BY second
ORDER BY second DESC
LIMIT 20;
```

---

## 持续集成 (CI) 配置

### GitHub Actions 示例

```yaml
# .github/workflows/test.yml
name: Run Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: windows-latest

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: test_password
        ports:
          - 5432:5432
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
      nats:
        image: nats:latest
        ports:
          - 4222:4222

    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run tests
        run: |
          cd codeswitch
          go test ./services/... -v -race -coverprofile=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./codeswitch/coverage.out
```

---

## 故障排查

### 问题 1: 测试无法找到包

**错误:** `package codeswitch/services/testdata is not in GOROOT`

**解决:**
```powershell
cd D:\tools\lurus-switch\codeswitch
go mod tidy
go mod download
```

### 问题 2: 数据库连接失败

**错误:** `dial tcp [::1]:5432: connect: connection refused`

**解决:**
```powershell
# 检查 PostgreSQL 是否运行
docker ps | findstr postgres

# 启动 PostgreSQL
docker-compose up -d postgres

# 查看日志
docker logs codeswitch-postgres
```

### 问题 3: 测试超时

**错误:** `panic: test timed out after 2m0s`

**解决:**
```powershell
# 增加超时时间
go test ./services/... -v -timeout 10m

# 或跳过慢速测试
go test ./services/... -v -short
```

### 问题 4: 竞态条件检测

**错误:** `WARNING: DATA RACE`

**解决:**
- 这表示代码存在并发访问问题
- 检查 Race Detector 输出中的文件和行号
- 使用互斥锁 (`sync.Mutex`) 或原子操作 (`sync/atomic`) 保护共享变量

---

## 测试覆盖率目标

| 模块 | 目标覆盖率 | 当前状态 |
|------|-----------|----------|
| `providerrelay.go` | 70% | 🔄 待测试 |
| `providerservice.go` | 80% | ✅ 已有测试 |
| `logservice.go` | 75% | 🔄 待测试 |
| `sync_integration.go` | 60% | ⏳ 集成测试阶段 |
| `billing_integration.go` | 60% | ⏳ 集成测试阶段 |

**查看覆盖率:**
```powershell
go test ./services/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

## 下一步

完成单元测试后,参考以下文档进行后续测试:

1. **完整测试计划**: `D:\tools\lurus-switch\doc\plans\lovely-stargazing-bachman.md`
2. **快速测试指南**: `D:\tools\lurus-switch\TESTING-QUICK-START.md`
3. **架构文档**: `D:\tools\lurus-switch\codeswitch\CLAUDE.md`

---

**创建时间**: 2026-01-12
**维护者**: CodeSwitch 测试团队
**版本**: v1.0
