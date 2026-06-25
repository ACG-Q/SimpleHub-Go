# DoneHub 类型

DoneHub 与 NewAPI 共享 tokens 代理路径，但分组和定价路径不同，且**不支持签到**。

## 认证方式

- `Authorization: Bearer <apiKey>`
- 无需额外用户标头

## API 代理 (`internal/handler/site_handler.go`)

| 接口 | 路径 | 行号 | 说明 |
|------|------|------|------|
| tokens | `/api/token/` | 1015/1037/1092/1108/1134 | 与 NewAPI/Veloera 相同 |
| 分组 | `/api/user_group_map` | 1156 | **与 NewAPI 不同**（NewAPI 用 `/api/user/self/groups`） |
| 定价 | `/api/available_model` | 1188 | **与 NewAPI 不同**（NewAPI 用 `/api/pricing`） |
| 兑换 | `/api/user/topup` | 1207 | 同 NewAPI |

**DoneHub 没有独立的转换函数**——代理请求/响应直接透传。

## 检测引擎 (`internal/checker/runner.go`)

| 操作 | 端点 | 行号 | 说明 |
|------|------|------|------|
| 模型获取 | `GET /api/available_model` | 337-344 | 仅 Bearer，无用户标头 |
| 账单查询 | `GET /api/user/self` | 543-623 | 解析 `data.quota`/`data.used_quota`，÷500000；无用户标头 |
| 签到 | — | — | **不支持** |
| 模型解析 | `parseDonehubModels(body)` | 1055-1081 | 格式：`{data:{model_id:{owned_by:"..."}}}`；`ownedBy` 回退 `"unknown"` |

## 关键函数一览

```
internal/handler/site_handler.go
└── proxyRequest.pathByType  → donehub 路由

internal/checker/runner.go
├── fetchModels()          → 337-344
├── fetchBillingDonehub   → 543-623
└── parseDonehubModels    → 1055-1081
```
