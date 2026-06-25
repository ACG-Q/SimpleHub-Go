# NewAPI 类型

NewAPI 是最常用的 API 聚合面板类型，也是 veloera / donehub 的代码基础。

## 认证方式

- `Authorization: Bearer <apiKey>`
- 额外标头 `New-Api-User: <userId>`

## API 代理 (`internal/handler/site_handler.go`)

| 接口 | 方法 | 路径 | 行号 |
|------|------|------|------|
| 令牌列表 | GET | `/api/token/` | 1013 |
| 创建令牌 | POST | `/api/token/` | 1035 |
| 更新令牌 | PUT | `/api/token/` | 1090 |
| 删除令牌 | DELETE | `/api/token/:tokenId` | 1106 |
| 获取令牌 Key | POST | `/api/token/:tokenId/key` | 1132 |
| 分组列表 | GET | `/api/user/self/groups` | 1154 |
| 定价信息 | GET | `/api/pricing` | 1186 |
| 兑换码 | POST | `/api/user/topup` | 1205 |

### 请求/响应转换

| 转换函数 | 行号 | 作用 |
|----------|------|------|
| `transformNewapiCreateRequest` | 638-670 | 创建令牌时将 camelCase 请求体转为 snake_case（`expiredTime`→`expired_time`），添加 `remain_amount: 0` 和 `cross_group_retry: false` |
| `transformNewapiTokens` | 672-706 | 列表响应从 `{"success":true,"data":{"items":[...]}}` 提取 tokens，key 添加 `sk-` 前缀 |
| `transformNewapiTokenKey` | 708-720 | 单个 key 响应从 `{"data":{"key":"..."}}` 提取并添加 `sk-` 前缀 |

## 检测引擎 (`internal/checker/runner.go`)

| 操作 | 端点 | 行号 | 说明 |
|------|------|------|------|
| 模型获取 | `GET /api/user/models` | 321-335 | `Authorization: Bearer` + `New-Api-User` 标头 |
| 账单查询 | `GET /api/user/self` | 449-541 | 解析 `data.quota` / `data.used_quota`，除以比率 500000 转为美元 |
| 签到 | `POST /api/user/checkin` | 167-176 | 需要 `UserID`；Bearer 认证 + `new-api-user` 标头 |
| 模型解析 | `parseNewapiModels(body, "newapi")` | 988-1053 | `ownedBy = "new-api"`；支持对象数组和字符串 ID 数组 |

## 关键函数一览

```
internal/handler/site_handler.go
├── doProxy()                    → 550-567 设置 New-Api-User 标头
├── proxyRequest.pathByType       → 按 newapi 路由到 /api/token/
├── transformNewapiCreateRequest → 638-670 请求体转换
├── transformNewapiTokens        → 672-706 响应列表转换
└── transformNewapiTokenKey      → 708-720 单个 key 转换

internal/checker/runner.go
├── Run()              → 92-96  启用签到、校验 UserID
├── fetchModels()      → 321-335 模型获取
├── fetchBillingNewapi → 449-541 账单查询
├── performCheckIn()   → 167-176 签到
└── parseNewapiModels  → 988-1053 模型解析
```
