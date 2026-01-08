# Lurus 多客户端同步 API 文档 / Multi-Client Sync API Documentation

> 版本: v1.0.0 | 更新日期: 2026-01-08

本文档面向客户端开发团队，描述如何实现多设备账户同步、配额实时更新等功能。

This document is for client development teams, describing how to implement multi-device account sync, real-time quota updates, and related features.

---

## 目录 / Table of Contents

1. [概述 / Overview](#概述--overview)
2. [服务端点 / Service Endpoints](#服务端点--service-endpoints)
3. [HTTP REST API](#http-rest-api)
4. [SSE 实时流 / SSE Real-time Stream](#sse-实时流--sse-real-time-stream)
5. [WebSocket 实时同步 / WebSocket Real-time Sync](#websocket-实时同步--websocket-real-time-sync)
6. [NATS 事件类型 / NATS Event Types](#nats-事件类型--nats-event-types)
7. [数据结构 / Data Structures](#数据结构--data-structures)
8. [错误处理 / Error Handling](#错误处理--error-handling)
9. [最佳实践 / Best Practices](#最佳实践--best-practices)
10. [示例代码 / Code Examples](#示例代码--code-examples)

---

## 概述 / Overview

Lurus 平台支持多客户端同时登录同一账户，所有客户端实时同步以下信息：

- **配额 (Quota)**: Token 使用量、剩余配额
- **余额 (Balance)**: 账户余额变动
- **权限 (Permission)**: 是否允许继续使用服务

### 架构图 / Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Mobile App     │    │  Desktop App    │    │  Web Client     │
│  (Flutter)      │    │  (Electron)     │    │  (Vue/React)    │
└────────┬────────┘    └────────┬────────┘    └────────┬────────┘
         │                      │                      │
         │         选择一种同步方式 / Choose one:       │
         │         ┌────────────────────────┐         │
         │         │ 1. HTTP 轮询 (简单)     │         │
         │         │ 2. SSE 流 (推荐)        │         │
         │         │ 3. WebSocket (全双工)   │         │
         │         └────────────────────────┘         │
         └──────────────────────┼──────────────────────┘
                                │
                   ┌────────────▼────────────┐
                   │   API Gateway           │
                   │   api.lurus.cn          │
                   └────────────┬────────────┘
                                │
                   ┌────────────▼────────────┐
                   │   Billing Service       │
                   │   配额管理 + 事件发布    │
                   └────────────┬────────────┘
                                │
                   ┌────────────▼────────────┐
                   │   NATS Message Bus      │
                   │   实时事件广播           │
                   └─────────────────────────┘
```

---

## 服务端点 / Service Endpoints

### 生产环境 / Production

| 服务 | 地址 | 用途 |
|------|------|------|
| API Gateway | `https://api.lurus.cn` | LLM API 调用 |
| Billing API | `https://api.lurus.cn/billing` | 配额/余额查询 |
| Sync WebSocket | `wss://api.lurus.cn/ws` | 实时同步 |

### 开发环境 / Development

| 服务 | 地址 | 用途 |
|------|------|------|
| Billing Service | `http://localhost:18103` | 本地测试 |
| Sync Service | `http://localhost:8081` | WebSocket 测试 |

---

## HTTP REST API

### 1. 获取同步状态 / Get Sync Status

获取用户当前的配额、余额等状态信息。

**请求 / Request**

```http
GET /api/v1/billing/sync/{user_id}
Authorization: Bearer {token}
```

**响应 / Response**

```json
{
  "user_id": "user_12345",
  "quota_limit": 1000000,
  "quota_used": 150000,
  "quota_remaining": 850000,
  "balance": 99.50,
  "allowed": true,
  "sync_time": "2026-01-08T14:18:57Z",
  "ttl": 30
}
```

**字段说明 / Field Description**

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | string | 用户 ID |
| `quota_limit` | int64 | 配额上限 (tokens) |
| `quota_used` | int64 | 已使用配额 |
| `quota_remaining` | int64 | 剩余配额 |
| `balance` | float64 | 账户余额 (USD) |
| `allowed` | bool | 是否允许继续使用 |
| `sync_time` | string | 同步时间 (RFC3339) |
| `ttl` | int | 建议刷新间隔 (秒) |

### 2. 检查余额 / Check Balance

快速检查用户是否有足够余额/配额继续使用。

**请求 / Request**

```http
GET /api/v1/billing/check/{user_id}
Authorization: Bearer {token}
```

**响应 / Response**

```json
{
  "user_id": "user_12345",
  "allowed": true,
  "balance": 99.50,
  "quota_limit": 1000000,
  "quota_used": 150000,
  "quota_remaining": 850000,
  "reason": ""
}
```

### 3. 获取配额详情 / Get Quota Details

**请求 / Request**

```http
GET /api/v1/billing/quota/{user_id}
Authorization: Bearer {token}
```

**响应 / Response**

```json
{
  "user_id": "user_12345",
  "quota_limit": 1000000,
  "quota_used": 150000,
  "quota_remaining": 850000
}
```

### 4. 获取使用统计 / Get Usage Statistics

**请求 / Request**

```http
GET /api/v1/billing/stats/{user_id}?start={timestamp}&end={timestamp}
Authorization: Bearer {token}
```

**参数 / Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start` | string | 否 | 开始时间 (RFC3339 或 Unix 时间戳) |
| `end` | string | 否 | 结束时间 (默认当前时间) |

**响应 / Response**

```json
{
  "user_id": "user_12345",
  "period_start": "2026-01-01T00:00:00Z",
  "period_end": "2026-01-08T23:59:59Z",
  "total_requests": 1250,
  "total_input_tokens": 125000,
  "total_output_tokens": 75000,
  "total_cost": 15.50,
  "by_model": {
    "claude-sonnet-4": {
      "requests": 800,
      "input_tokens": 80000,
      "output_tokens": 50000,
      "cost": 10.00
    },
    "gpt-4o": {
      "requests": 450,
      "input_tokens": 45000,
      "output_tokens": 25000,
      "cost": 5.50
    }
  }
}
```

---

## SSE 实时流 / SSE Real-time Stream

Server-Sent Events (SSE) 提供单向实时推送，适合大多数场景。

### 连接 / Connect

```http
GET /api/v1/billing/sync/{user_id}/stream
Authorization: Bearer {token}
Accept: text/event-stream
```

### 事件格式 / Event Format

```
event: message
data: {"type":"sync","quota_limit":1000000,"quota_used":150000,"quota_remaining":850000,"balance":99.50,"timestamp":"2026-01-08T14:18:57Z"}

event: message
data: {"type":"heartbeat","quota_remaining":850000,"balance":99.50,"timestamp":"2026-01-08T14:19:27Z"}
```

### 事件类型 / Event Types

| type | 说明 | 触发条件 |
|------|------|---------|
| `sync` | 完整同步数据 | 连接建立时 |
| `heartbeat` | 心跳 + 状态更新 | 每 30 秒 |
| `quota_updated` | 配额变更 | 使用 API 后 |
| `balance_changed` | 余额变动 | 充值/扣费 |
| `quota_low` | 低配额警告 | 配额 ≥80% 使用 |
| `quota_exhausted` | 配额耗尽 | 配额 100% 使用 |

### 客户端实现 / Client Implementation

```javascript
// JavaScript/TypeScript
const eventSource = new EventSource(
  'https://api.lurus.cn/api/v1/billing/sync/user_12345/stream',
  { headers: { 'Authorization': 'Bearer ' + token } }
);

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Sync event:', data.type, data);

  switch (data.type) {
    case 'sync':
    case 'heartbeat':
      updateQuotaDisplay(data.quota_remaining, data.balance);
      break;
    case 'quota_low':
      showWarning('配额即将用尽');
      break;
    case 'quota_exhausted':
      showError('配额已用尽');
      disableFeatures();
      break;
  }
};

eventSource.onerror = (error) => {
  console.error('SSE error:', error);
  // 自动重连由浏览器处理
};
```

---

## WebSocket 实时同步 / WebSocket Real-time Sync

WebSocket 提供全双工通信，支持客户端主动请求。

### 连接 / Connect

```
wss://api.lurus.cn/ws/sync
```

**连接参数 (Query String)**

| 参数 | 必填 | 说明 |
|------|------|------|
| `token` | 是 | 认证 Token |
| `device_id` | 否 | 设备标识 (用于多设备管理) |

**示例 / Example**

```
wss://api.lurus.cn/ws/sync?token=Bearer%20xxx&device_id=mobile_001
```

### 消息格式 / Message Format

所有消息均为 JSON 格式：

```json
{
  "type": "message_type",
  "user_id": "user_12345",
  "device_id": "mobile_001",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": { ... }
}
```

### 客户端 → 服务器消息 / Client to Server

#### ping - 心跳检测

```json
{
  "type": "ping"
}
```

响应:

```json
{
  "type": "pong",
  "timestamp": "2026-01-08T14:18:57Z"
}
```

#### sync_request - 请求同步

```json
{
  "type": "sync_request"
}
```

响应: 服务器推送完整同步数据

#### subscribe - 订阅频道

```json
{
  "type": "subscribe",
  "data": {
    "channels": ["billing", "notifications"]
  }
}
```

### 服务器 → 客户端消息 / Server to Client

#### nats - NATS 事件转发

```json
{
  "type": "nats",
  "user_id": "user_12345",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": {
    "type": "quota.updated",
    "quota_limit": 1000000,
    "quota_used": 160000,
    "quota_remaining": 840000,
    "percent_used": 16.0
  }
}
```

### 客户端实现 / Client Implementation

```typescript
// TypeScript
class SyncClient {
  private ws: WebSocket;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  connect(token: string, deviceId: string) {
    const url = `wss://api.lurus.cn/ws/sync?token=${encodeURIComponent(token)}&device_id=${deviceId}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
      this.requestSync();
    };

    this.ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      this.handleMessage(msg);
    };

    this.ws.onclose = () => {
      this.scheduleReconnect();
    };
  }

  private handleMessage(msg: any) {
    switch (msg.type) {
      case 'pong':
        // 心跳响应
        break;
      case 'nats':
        this.handleNatsEvent(msg.data);
        break;
    }
  }

  private handleNatsEvent(data: any) {
    switch (data.type) {
      case 'quota.updated':
        this.onQuotaUpdated?.(data);
        break;
      case 'balance.changed':
        this.onBalanceChanged?.(data);
        break;
      case 'quota.low':
        this.onQuotaLow?.(data);
        break;
      case 'quota.exhausted':
        this.onQuotaExhausted?.(data);
        break;
    }
  }

  requestSync() {
    this.send({ type: 'sync_request' });
  }

  ping() {
    this.send({ type: 'ping' });
  }

  private send(data: any) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
      setTimeout(() => this.connect(), delay);
      this.reconnectAttempts++;
    }
  }

  // 事件回调
  onQuotaUpdated?: (data: any) => void;
  onBalanceChanged?: (data: any) => void;
  onQuotaLow?: (data: any) => void;
  onQuotaExhausted?: (data: any) => void;
}
```

---

## NATS 事件类型 / NATS Event Types

NATS 事件通过 SSE 或 WebSocket 转发给客户端。

### 事件主题 / Event Subjects

| 主题模式 | 说明 |
|---------|------|
| `billing.{user_id}` | 用户计费事件 |
| `user.{user_id}.*` | 用户相关事件 |
| `chat.{user_id}.>` | 聊天会话事件 |

### quota.updated - 配额更新

```json
{
  "type": "quota.updated",
  "user_id": "user_12345",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": {
    "quota_limit": 1000000,
    "quota_used": 160000,
    "quota_remaining": 840000,
    "percent_used": 16.0
  }
}
```

### balance.changed - 余额变动

```json
{
  "type": "balance.changed",
  "user_id": "user_12345",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": {
    "balance": 89.50,
    "change": -10.00,
    "reason": "api_usage",
    "reference_id": "txn_abc123"
  }
}
```

### usage.recorded - 使用记录

```json
{
  "type": "usage.recorded",
  "user_id": "user_12345",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": {
    "platform": "claude",
    "model": "claude-sonnet-4",
    "input_tokens": 1500,
    "output_tokens": 800,
    "cost": 0.05,
    "trace_id": "req_xyz789"
  }
}
```

### quota.low - 低配额警告

```json
{
  "type": "quota.low",
  "user_id": "user_12345",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": {
    "remaining": 150000,
    "percent_used": 85.0,
    "message": "Quota is 85.0% used, 150000 tokens remaining"
  }
}
```

### quota.exhausted - 配额耗尽

```json
{
  "type": "quota.exhausted",
  "user_id": "user_12345",
  "timestamp": "2026-01-08T14:18:57Z",
  "data": {
    "message": "Quota exhausted. Please upgrade or wait for reset."
  }
}
```

---

## 数据结构 / Data Structures

### SyncStatus

```typescript
interface SyncStatus {
  user_id: string;
  quota_limit: number;      // 配额上限
  quota_used: number;       // 已使用
  quota_remaining: number;  // 剩余
  balance: number;          // 余额 (USD)
  allowed: boolean;         // 是否允许使用
  sync_time: string;        // ISO 8601 时间
  ttl: number;              // 建议刷新间隔 (秒)
}
```

### QuotaUpdateData

```typescript
interface QuotaUpdateData {
  quota_limit: number;
  quota_used: number;
  quota_remaining: number;
  percent_used: number;     // 0-100
}
```

### BalanceChangeData

```typescript
interface BalanceChangeData {
  balance: number;          // 新余额
  change: number;           // 变动金额 (正数增加，负数减少)
  reason: string;           // 原因: api_usage, recharge, refund, adjustment
  reference_id?: string;    // 关联交易 ID
}
```

### UsageRecordData

```typescript
interface UsageRecordData {
  platform: string;         // claude, openai, gemini
  model: string;            // 模型名称
  input_tokens: number;
  output_tokens: number;
  cost: number;             // USD
  trace_id?: string;        // 请求追踪 ID
}
```

---

## 错误处理 / Error Handling

### HTTP 错误码 / HTTP Error Codes

| 状态码 | 说明 | 处理方式 |
|--------|------|---------|
| 400 | 请求参数错误 | 检查请求格式 |
| 401 | 未认证 | 刷新 Token |
| 403 | 无权限 | 检查用户权限 |
| 404 | 用户不存在 | 引导用户注册 |
| 429 | 请求过于频繁 | 实现退避重试 |
| 500 | 服务器错误 | 重试或联系支持 |

### 错误响应格式 / Error Response Format

```json
{
  "error": "user not found",
  "code": "USER_NOT_FOUND",
  "details": "User with ID user_12345 does not exist"
}
```

### 重试策略 / Retry Strategy

```typescript
async function fetchWithRetry(url: string, options: RequestInit, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(url, options);
      if (response.status === 429) {
        const retryAfter = response.headers.get('Retry-After') || '1';
        await sleep(parseInt(retryAfter) * 1000);
        continue;
      }
      return response;
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await sleep(Math.pow(2, i) * 1000); // 指数退避
    }
  }
}
```

---

## 最佳实践 / Best Practices

### 1. 选择合适的同步方式

| 场景 | 推荐方式 | 原因 |
|------|---------|------|
| Web 应用 | SSE | 简单、自动重连 |
| 移动应用 | WebSocket | 双向通信、省电 |
| 后台服务 | HTTP 轮询 | 最简单、无状态 |
| 实时性要求极高 | WebSocket | 最低延迟 |

### 2. 实现本地缓存

```typescript
class QuotaCache {
  private cache: SyncStatus | null = null;
  private lastUpdate: number = 0;
  private ttl: number = 30000; // 30秒

  get(): SyncStatus | null {
    if (Date.now() - this.lastUpdate > this.ttl) {
      return null; // 缓存过期
    }
    return this.cache;
  }

  set(data: SyncStatus) {
    this.cache = data;
    this.lastUpdate = Date.now();
    this.ttl = data.ttl * 1000;
  }
}
```

### 3. 优雅降级

```typescript
async function getQuotaStatus(userId: string): Promise<SyncStatus> {
  // 1. 先检查本地缓存
  const cached = quotaCache.get();
  if (cached) return cached;

  // 2. 尝试 SSE/WebSocket 获取
  if (syncClient.isConnected()) {
    return syncClient.getLastStatus();
  }

  // 3. 回退到 HTTP API
  return await fetchSyncStatus(userId);
}
```

### 4. 处理配额警告

```typescript
function handleQuotaWarning(data: QuotaUpdateData) {
  const percentUsed = data.percent_used;

  if (percentUsed >= 100) {
    // 禁用功能
    disableAIFeatures();
    showUpgradeDialog();
  } else if (percentUsed >= 90) {
    // 紧急警告
    showUrgentWarning(`仅剩 ${data.quota_remaining} tokens`);
  } else if (percentUsed >= 80) {
    // 普通提醒
    showNotification(`配额已使用 ${percentUsed}%`);
  }
}
```

### 5. 多设备冲突处理

当同一用户在多个设备上同时使用时：

```typescript
// 收到其他设备的使用记录时
function handleRemoteUsage(data: UsageRecordData) {
  // 更新本地显示
  updateQuotaDisplay();

  // 可选：提示用户
  if (data.trace_id !== currentRequestId) {
    showNotification('其他设备正在使用 AI 服务');
  }
}
```

---

## 示例代码 / Code Examples

### Flutter (Dart)

```dart
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

class SyncService {
  WebSocketChannel? _channel;

  void connect(String token, String userId) {
    final uri = Uri.parse(
      'wss://api.lurus.cn/ws/sync?token=$token&device_id=flutter_app'
    );
    _channel = WebSocketChannel.connect(uri);

    _channel!.stream.listen((message) {
      final data = jsonDecode(message);
      _handleMessage(data);
    });
  }

  void _handleMessage(Map<String, dynamic> data) {
    switch (data['type']) {
      case 'nats':
        final event = data['data'];
        switch (event['type']) {
          case 'quota.updated':
            onQuotaUpdated?.call(event);
            break;
          case 'quota.low':
            onQuotaLow?.call(event);
            break;
        }
        break;
    }
  }

  Function(Map<String, dynamic>)? onQuotaUpdated;
  Function(Map<String, dynamic>)? onQuotaLow;
}
```

### Swift (iOS)

```swift
import Foundation

class SyncClient: NSObject, URLSessionWebSocketDelegate {
    private var webSocket: URLSessionWebSocketTask?

    func connect(token: String, userId: String) {
        let url = URL(string: "wss://api.lurus.cn/ws/sync?token=\(token)&device_id=ios_app")!
        let session = URLSession(configuration: .default, delegate: self, delegateQueue: nil)
        webSocket = session.webSocketTask(with: url)
        webSocket?.resume()
        receiveMessage()
    }

    private func receiveMessage() {
        webSocket?.receive { [weak self] result in
            switch result {
            case .success(let message):
                if case .string(let text) = message,
                   let data = text.data(using: .utf8),
                   let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                    self?.handleMessage(json)
                }
                self?.receiveMessage()
            case .failure(let error):
                print("WebSocket error: \(error)")
            }
        }
    }

    private func handleMessage(_ data: [String: Any]) {
        guard let type = data["type"] as? String else { return }
        // Handle message...
    }
}
```

### Kotlin (Android)

```kotlin
import okhttp3.*
import org.json.JSONObject

class SyncClient(private val token: String) {
    private var webSocket: WebSocket? = null
    private val client = OkHttpClient()

    fun connect() {
        val request = Request.Builder()
            .url("wss://api.lurus.cn/ws/sync?token=$token&device_id=android_app")
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                val json = JSONObject(text)
                handleMessage(json)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                // Handle error, schedule reconnect
            }
        })
    }

    private fun handleMessage(data: JSONObject) {
        when (data.getString("type")) {
            "nats" -> {
                val event = data.getJSONObject("data")
                when (event.getString("type")) {
                    "quota.updated" -> onQuotaUpdated?.invoke(event)
                    "quota.low" -> onQuotaLow?.invoke(event)
                }
            }
        }
    }

    var onQuotaUpdated: ((JSONObject) -> Unit)? = null
    var onQuotaLow: ((JSONObject) -> Unit)? = null
}
```

---

## 联系方式 / Contact

如有问题，请联系：

- **技术支持**: tech@lurus.cn
- **API 问题**: api-support@lurus.cn
- **文档反馈**: docs@lurus.cn

---

## 变更日志 / Changelog

### v1.0.0 (2026-01-08)

- 初始版本
- HTTP REST API: sync, check, quota, stats
- SSE 实时流支持
- WebSocket 双向通信
- NATS 事件: quota.updated, balance.changed, usage.recorded, quota.low, quota.exhausted
