# SimpleHub REST API 文档

## 通用规范

| 项目 | 说明 |
|------|------|
| 字段命名 | 所有 JSON key 使用小驼峰 (camelCase) |
| 认证 | `Authorization: Bearer <JWT>`（除 `/api/auth/login`） |
| 基础 URL | `http://<host>:<port>/<security-entry>` （安全入口为首次运行时生成的随机 8 位 hex 路径） |
| 内容类型 | `application/json` |

---

## 目录

- [Auth](#auth)
- [Sites](#sites)
  - [CRUD](#sites-crud)
  - [检测](#sites-check)
  - [令牌代理](#sites-tokens-proxy)
  - [快照与差异](#sites-snapshots--diffs)
- [导出/导入](#export--import)
- [Categories](#categories)
- [Email Config](#email-config)
- [Schedule Config](#schedule-config)

---

## Auth

### POST /api/auth/login

管理员登录，获取 JWT token。

**Request Body:**

```json
{
  "email": "string (required)",
  "password": "string (required)"
}
```

**Response 200:**

```json
{ "token": "string" }
```

**Response 401:**

```json
{ "error": "管理员账号或密码错误" }
```

---

## Sites

### Sites CRUD

#### GET /api/sites

获取站点列表（支持搜索）。返回每个站点关联的最新快照数据（额度、签到状态等）。

**Query Parameters:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| search | string | 否 | 按名称/URL/模型 ID 搜索 |

**Response 200:** `Site[]`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | |
| name | string | |
| baseUrl | string | |
| apiType | string | `newapi`, `veloera`, `donehub`, `voapi`, `other` |
| userId | string? | |
| billingUrl | string? | |
| billingAuthType | string | `token` / `cookie` |
| billingLimitField | string? | |
| billingUsageField | string? | |
| unlimitedQuota | boolean | |
| enableCheckIn | boolean | |
| checkInMode | string | `model` / `checkin` / `both` |
| scheduleCron | string? | |
| timezone | string | |
| pinned | boolean | |
| excludeFromBatch | boolean | |
| categoryId | string? | |
| extralink | string? | |
| remark | string? | |
| sortOrder | integer | |
| lastCheckedAt | datetime? | |
| createdAt | datetime | |
| category | Category? | |
| billingLimit | number? | 来自最新快照 |
| billingUsage | number? | 来自最新快照 |
| billingError | string? | 来自最新快照 |
| checkInSuccess | boolean? | 来自最新快照 |
| checkInMessage | string? | 来自最新快照 |
| checkInError | string? | 来自最新快照 |

---

#### POST /api/sites

创建站点。

**Request Body:**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| name | string | 是 | — | |
| baseUrl | string | 是 | — | |
| apiKey | string | 是 | — | 明文，服务端加密存储 |
| apiType | string | 否 | `other` | `newapi`, `veloera`, `donehub`, `voapi`, `other` |
| userId | string | 否 | null | |
| billingUrl | string | 否 | null | |
| billingAuthType | string | 否 | `token` | `token`, `cookie` |
| billingAuthValue | string | 否 | null | 明文，服务端加密存储 |
| proxyUrl | string | 否 | null | 明文，服务端加密存储 |
| billingLimitField | string | 否 | null | |
| billingUsageField | string | 否 | null | |
| unlimitedQuota | boolean | 否 | false | |
| enableCheckIn | boolean | 否 | false | |
| checkInMode | string | 否 | `both` | `model`, `checkin`, `both` |
| scheduleCron | string | 否 | null | |
| timezone | string | 否 | `UTC` | |
| pinned | boolean | 否 | false | |
| excludeFromBatch | boolean | 否 | false | |
| categoryId | string | 否 | null | |
| extralink | string | 否 | null | |
| remark | string | 否 | null | |
| sortOrder | integer | 否 | 0 | |

**Response 201:** Site（同 List 字段，不含加密存储字段）

**Response 400:**

```json
{ "error": "请提供必填字段" }
```

---

#### GET /api/sites/:id

获取站点详情（解密敏感字段）。

**Response 200:** 继承 Site 所有字段，额外追加：

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string? | 解密后的 apiKey |
| type | string | apiType 的别名 |
| proxyUrl | string? | 解密后的代理 URL |
| billingAuthValue | string? | 解密后的额度认证值 |

**Response 404:**

```json
{ "error": "站点不存在" }
```

---

#### PATCH /api/sites/:id

更新站点（部分更新，未传字段保持原值）。

布尔字段即使传 false 也会更新；可空字段传 null 会置空；所有字段可选。
请求字段同 Create，所有字段 optional。

**Response 200:**

```json
{ "success": true }
```

**Response 400:**

```json
{ "error": "..." }
```

---

#### DELETE /api/sites/:id

删除站点（级联删除快照/差异）。

**Response 200:**

```json
{ "success": true }
```

---

#### POST /api/sites/reorder

批量更新站点排序。

**Request Body:**

```json
{
  "orders": [
    { "id": "string", "sortOrder": "integer" }
  ]
}
```

**Response 200:**

```json
{ "success": true }
```

---

### Sites Check

#### POST /api/sites/:id/check

手动检测站点。

**Query Parameters:**

| 参数 | 类型 | 说明 |
|------|------|------|
| skipNotification | string | `"true"` 跳过通知 |

**Response 200:** CheckResult

```json
{
  "hasChanges": "boolean",
  "models": "ModelInfo[]",
  "billingLimit": "number?",
  "billingUsage": "number?",
  "checkInSuccess": "boolean?",
  "checkInQuota": "number?"
}
```

**Response 500:**

```json
{ "error": "检测失败: ..." }
```

---

### Sites Snapshots & Diffs

#### GET /api/sites/:id/snapshots

获取站点快照列表（仅成功快照）。

| 参数 | 类型 | 说明 |
|------|------|------|
| limit | integer | 数量限制（默认 1） |

**Response 200:** `ModelSnapshot[]`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | |
| siteId | string | |
| fetchedAt | datetime | |
| modelsJson | array | 解析后的模型列表 |
| hash | string | SHA256(modelsJson) |
| rawResponse | string? | |
| errorMessage | string? | |
| statusCode | integer? | |
| responseTime | integer? | ms |
| billingLimit | number? | |
| billingUsage | number? | |
| billingError | string? | |
| checkInSuccess | boolean? | |
| checkInMessage | string? | |
| checkInQuota | number? | |
| checkInError | string? | |

---

#### GET /api/sites/:id/latest-snapshot

获取站点最新快照（含错误快照）。

**Response 200:** ModelSnapshot

**Response 404:**

```json
{ "error": "无快照数据" }
```

---

#### GET /api/sites/:id/diffs

获取站点模型差异列表。

| 参数 | 类型 | 说明 |
|------|------|------|
| limit | integer | 数量限制（默认 50） |

**Response 200:** `ModelDiff[]`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | |
| siteId | string | |
| diffAt | datetime | |
| addedJson | array | 新增模型列表 |
| removedJson | array | 移除模型列表 |
| changedJson | array | 变更模型列表 |
| snapshotFromId | string? | |
| snapshotToId | string? | |

---

### Sites Tokens (Proxy)

以下端点代理到上游 API，按站点 `apiType` 自动路由。

#### GET /api/sites/:id/tokens
#### POST /api/sites/:id/tokens
#### PUT /api/sites/:id/tokens
#### DELETE /api/sites/:id/tokens/:tokenId
#### POST /api/sites/:id/tokens/:tokenId/key
#### GET /api/sites/:id/groups
#### GET /api/sites/:id/pricing
#### POST /api/sites/:id/redeem

---

## Export / Import

### GET /api/sites/export

导出所有站点和分类（加密字段已解密）。

**Response 200:**

```json
{
  "version": "string",
  "exportDate": "string (ISO datetime)",
  "categories": "Category[]",
  "sites": "ExportedSite[]"
}
```

---

### GET /api/exports/sites

导出别名，同 `/api/sites/export`。

---

### POST /api/sites/import

导入站点和分类。

**Request Body:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| version | string | 否 | |
| exportDate | string | 否 | |
| categories | Category[] | 否 | 每个含 name/scheduleCron/timezone |
| sites | ExportedSite[] | 是 | 见下方 |

**ExportedSite 字段:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | |
| baseUrl | string | 是 | |
| apiKey | string | 是 | |
| apiType | string | 否 | |
| userId | string? | 否 | |
| scheduleCron | string? | 否 | |
| timezone | string | 否 | |
| pinned | boolean | 否 | |
| excludeFromBatch | boolean | 否 | |
| billingUrl | string? | 否 | |
| billingAuthType | string | 否 | |
| billingAuthValue | string? | 否 | |
| proxyUrl | string? | 否 | |
| billingLimitField | string? | 否 | |
| billingUsageField | string? | 否 | |
| unlimitedQuota | boolean | 否 | |
| enableCheckIn | boolean | 否 | |
| checkInMode | string | 否 | |
| categoryName | string? | 否 | 按名称匹配/创建分类 |

**Response 200:**

```json
{
  "imported": "integer",
  "total": "integer",
  "errors?": "string[]"
}
```

**Response 400:**

```json
{ "error": "..." }
```

---

## Categories

### GET /api/categories

获取分类列表（含每个分类下的站点）。

**Response 200: `Category[]`**

| 字段 | 类型 |
|------|------|
| id | string |
| name | string |
| scheduleCron | string? |
| timezone | string |
| createdAt | datetime |
| updatedAt | datetime |
| sites | Site[] |

---

### POST /api/categories

创建分类。

**Request Body:**

```json
{
  "name": "string (required)",
  "scheduleCron": "string?",
  "timezone": "string (default: Asia/Shanghai)"
}
```

**Response 200:** Category

**Response 400:**

```json
{ "error": "分类名称已存在" }
```

---

### PATCH /api/categories/:id

更新分类。

**Request Body:**

```json
{
  "name": "string?",
  "scheduleCron": "string? (nullable)",
  "timezone": "string?"
}
```

**Response 200:** Category

---

### DELETE /api/categories/:id

删除分类（其下站点 categoryId 置 null）。

**Response 200:**

```json
{ "success": true }
```

---

### POST /api/categories/:id/check

一键检测分类下所有站点。

**Query Parameters:**

| 参数 | 类型 | 说明 |
|------|------|------|
| skipNotification | string | `"true"` 跳过通知 |

**Response 200:** CheckResult

---

## Email Config

### GET /api/email-config

获取邮件通知配置（不返回加密 API Key）。

**Response 200:**

```json
{
  "enabled": "boolean",
  "notifyEmails": "string (逗号分隔)"
}
```

---

### POST /api/email-config

创建/更新邮件通知配置。

**Request Body:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| resendApiKey | string | 是 | 明文，服务端加密存储 |
| notifyEmails | string | 是 | 逗号分隔的邮箱列表 |
| enabled | boolean | 否 | 默认 true |

**Response 200:** EmailConfig（不含 resendApiKeyEnc）

**Response 400:**

```json
{ "error": "邮箱格式不正确: xxx" }
```

---

### POST /api/email-config/test (*TEST*)

发送测试邮件到已配置的通知邮箱。

**Response 200:**

```json
{ "success": true, "message": "测试邮件已发送" }
```

**Response 500:**

```json
{ "error": "..." }
```

---

## Schedule Config

### GET /api/schedule-config

获取定时检测配置。

**Response 200:**

```json
{
  "ok": true,
  "config": {
    "id": "string",
    "enabled": "boolean",
    "hour": "integer (0-23)",
    "minute": "integer (0-59)",
    "timezone": "string",
    "interval": "integer (5-300s)",
    "overrideIndividual": "boolean",
    "lastRun": "datetime?",
    "createdAt": "datetime",
    "updatedAt": "datetime"
  }
}
```

---

### POST /api/schedule-config

更新定时检测配置。

**Request Body:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enabled | boolean | 是 | |
| hour | integer | 是 | 0-23 |
| minute | integer | 是 | 0-59 |
| interval | integer | 是 | 5-300 秒 |
| overrideIndividual | boolean | 否 | 默认 false |

**Response 200:**

```json
{
  "ok": true,
  "config": { "...ScheduleConfig..." }
}
```

**Response 400:**

```json
{ "error": "hour 必须在 0-23 之间" }
```

---

### POST /api/schedule-config/trigger (*TEST*)

立即触发全局检测（后台执行）。

**Response 200:**

```json
{ "success": true, "message": "已触发全局检测（后台执行）" }
```
