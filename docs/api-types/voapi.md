# VOAPI 类型

VOAPI 的认证方式、路径、数据格式与其他类型完全不同，拥有最多的转换函数。

## 认证方式

- `Authorization: <apiKey>`（**无** `Bearer` 前缀，即原始密钥）
- 无需用户标头（签到也不需要 `UserID`）

## API 代理 (`internal/handler/site_handler.go`)

| 接口 | 方法 | 路径 | 行号 |
|------|------|------|------|
| 令牌列表 | GET | `/api/keys` | 1016 |
| 创建令牌 | POST | `/api/keys` | 1038 |
| 更新令牌 | PUT | `/api/keys/:id`（直接处理，不走 doProxy） | 1057-1085 |
| 删除令牌 | DELETE | `/api/keys/:tokenId` | 1109 |
| 获取令牌 Key | — | 不支持，返回 400 | 1124-1126 |
| 分组列表 | GET | `/api/models` | 1157 |
| 定价信息 | GET | `/api/models`（额外 User-Agent + Accept） | 1189 |
| 兑换码 | POST | `/api/user/topup` | 1208 |

### authType = "voapi_raw"

在 `doProxy` 中设置（550-552）：
```go
authType: "voapi_raw",  // 发送原始 Key，无 Bearer 前缀
```

### 请求/响应转换

VOAPI 拥有最多、最复杂的转换函数：

| 转换函数 | 行号 | 作用 |
|----------|------|------|
| `transformVoapiTokens` | 722-816 | `{code:0, data:{records:[...]}}` → 标准化 tokens 列表。映射 voapi 字段：`token/key→key`，`expireTime→expired_time`，`boundlessAmount→unlimited_quota`，`amount*500000→remain_quota` |
| `transformVoapiCreateRequest` | 818-860 | 将标准请求转为 voapi 格式：`remainQuota/500000→amount`，处理 `-1` 过期时间 |
| `transformVoapiCreateResponse` | 862-876 | `{code:0,...}` → `{success:true,...}` |
| `transformVoapiUpdateRequest` | 878-926 | 同上，额外处理 `serializeManagedTokenKey`（去除 `sk-` 前缀） |
| `transformVoapiUpdateResponse` | 928-942 | 同创建响应 |
| `transformVoapiDeleteResponse` | 944-958 | 标准删除响应转换 |
| `transformVoapiGroups` | 960-984 | `{code:0, data:{groups:[{id,name}]}}` → `{success:true, data:{id:{name,desc}}}` |

### `UpdateToken` 特殊处理

VOAPI 的 `UpdateToken` 不走 `doProxy` 通用代理（1057-1085），而是**内联处理**：
1. 读取请求体，解析 `body.ID`
2. 构造 `PUT /api/keys/{id}`
3. 使用 `transformVoapiUpdateRequest` 和 `transformVoapiUpdateResponse`
4. 独立返回响应

## 检测引擎 (`internal/checker/runner.go`)

| 操作 | 端点 | 行号 | 说明 |
|------|------|------|------|
| 模型获取 | `GET /api/models` | 346-352 | 原始 Key 认证 |
| 账单查询 | `GET /api/user/info` | 626-709 | 解析 `bindBalance`+`basicBalance`，**不**除以 500000 |
| 签到 | `POST /api/check_in` | 187-195 | **不需要** `UserID`；原始 Key 认证；成功判断依据 `parsed.Code == 0` |
| 签到结果 | — | 263-278 | 从 `data.amount` 读取配额（非 `data.quota`） |
| 模型解析 | `parseVoapiModels(body)` | 1083-1174 | 格式：`{code:0, data:{models:[...]}}`；解析 `chargingType`、`inputPrice`、`outputPrice`、`singlePrice`；处理毫秒级 `created` 时间戳 |

## 关键函数一览

```
internal/handler/site_handler.go
├── doProxy()                  → 550-552 authType=voapi_raw
├── transformVoapiTokens       → 722-816
├── transformVoapiCreateRequest  → 818-860
├── transformVoapiCreateResponse → 862-876
├── transformVoapiUpdateRequest  → 878-926
├── transformVoapiUpdateResponse → 928-942
├── transformVoapiDeleteResponse → 944-958
├── transformVoapiGroups       → 960-984
├── UpdateToken (内联)         → 1057-1085
├── GetTokenKey (返回 400)     → 1124-1126
└── GetPricing (额外标头)      → 1179-1182

internal/checker/runner.go
├── fetchModels()         → 346-352
├── fetchBillingVoapi     → 626-709
├── performCheckIn()      → 187-195 (签到), 263-278 (解析)
└── parseVoapiModels      → 1083-1174
```
