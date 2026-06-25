# Veloera 类型

Veloera 与 NewAPI 共享大部分代码，仅在标头名称和签到端点上存在差异。

## 与 NewAPI 的差异

| 项目 | NewAPI | Veloera |
|------|--------|---------|
| 用户标头 | `New-Api-User` | `Veloera-User` |
| 签到端点 | `POST /api/user/checkin` | `POST /api/user/check_in` |
| 模型 `ownedBy` | `"new-api"` | `"veloera"` |

**除此之外，所有代理路径、转换函数、账单解析逻辑与 NewAPI 完全相同。**

## API 代理 (`internal/handler/site_handler.go`)

| 接口 | 路径 | 行号 |
|------|------|------|
| tokens | `/api/token/` | 1014-1036-1091-1107 |
| 分组 | `/api/user/self/groups` | 1155 |
| 定价 | `/api/pricing` | 1187 |
| 兑换 | `/api/user/topup` | 1206 |

### 差异代码

`doProxy()` 函数中根据 `site.APIType` 分支设置不同的用户标头：

```go
// site_handler.go:562-567
if site.APIType == "newapi" {
    proxy.extraHeaders["New-Api-User"] = *site.UserID
} else if site.APIType == "veloera" {
    proxy.extraHeaders["Veloera-User"] = *site.UserID
}
```

## 检测引擎 (`internal/checker/runner.go`)

| 操作 | 行号 | 说明 |
|------|------|------|
| 模型获取 | 321-333 | 通过 `fetchModels` 条件分支切换 `Veloera-User` 标头 |
| 账单查询 | 468-472 | `fetchBillingNewapi` 中切换用户标头（共享 NewAPI 账单逻辑） |
| 签到 | 177-186 | `performCheckIn` 使用 `/api/user/check_in` 和 `veloera-user` 标头 |
| 模型解析 | 1001-1003 | `parseNewapiModels(body, "veloera")`，`ownedBy = "veloera"` |

## 关键函数一览

```
internal/handler/site_handler.go
└── doProxy()            → 565-567 Veloera-User 标头

internal/checker/runner.go
├── fetchModels()        → 331-333 Veloera-User 标头
├── fetchBillingNewapi() → 468-472 Veloera-User 标头
├── performCheckIn()     → 177-186 /api/user/check_in
└── parseNewapiModels    → 1001-1003 ownedBy = "veloera"
```
