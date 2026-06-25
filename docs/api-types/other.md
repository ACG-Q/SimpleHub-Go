# Other 类型（通用/回退）

`other` 是站点的默认 API 类型，适用于 OpenAI 兼容接口或其他无法归类的服务。**不支持令牌管理、签到。**

## 认证方式

- `Authorization: Bearer <apiKey>`
- 无额外标头

## API 代理 (`internal/handler/site_handler.go`)

`other` 类型不支持 tokens / groups / pricing 等代理接口。`GetPricing` 直接返回错误：

```go
// site_handler.go:1173-1175
if not in [newapi, veloera, donehub, voapi] {
    return 400: "此站点类型不支持pricing接口"
}
```

## 检测引擎 (`internal/checker/runner.go`)

### 模型获取（354-359）

使用标准 OpenAI 兼容端点：

```
GET /v1/models
Authorization: Bearer <apiKey>
```

### 模型解析（1176-1244）

`parseOpenaiModels(body)`：

- 格式：`{"data":[{"id","object","owned_by","created",...}]}`
- 也支持纯字符串 ID 数组
- `ownedBy` 回退 `"unknown"`

### 账单查询（444-446, 712-958）

`fetchBilling()` 根据站点配置分两条路径：

#### 1. 自定义账单 URL（`fetchCustomBilling`, 722-840）

当站点配置了 `billingUrl` 时使用：

- 支持 `token` 认证（可带或不带 `Bearer` 前缀）
- 支持 `cookie` 认证
- 使用 `BillingLimitField` / `BillingUsageField` 自定义字段路径
- 解析嵌套 JSON：`limit` / `usage` / `quota` / `used` / `balance` / `consumed` / `system_hard_limit_usd` / `total_usage`

#### 2. OpenAI Billing（`fetchOpenAIBilling`, 842-958）

当未配置自定义账单 URL 时使用，**并行发送两个请求**：

| 请求 | 端点 | 返回值 |
|------|------|--------|
| 额度上限 | `GET /v1/dashboard/billing/subscription` | `system_hard_limit_usd` |
| 用量 | `GET /v1/dashboard/billing/usage` | `total_usage * 0.01` |

包含浏览器标头（`User-Agent`、`Accept-Language`、`Cache-Control`）。

### 签到

不支持（`Run()` 中签到条件排除 `other`）。

## 关键函数一览

```
internal/handler/site_handler.go
├── GetPricing() → 1173 返回错误

internal/checker/runner.go
├── fetchModels()         → 354-359
├── fetchBillingOther     → 712-720
├── fetchCustomBilling    → 722-840
├── fetchOpenAIBilling    → 842-958
└── parseOpenaiModels     → 1176-1244
```
