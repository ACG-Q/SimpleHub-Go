# API 类型代码位置指南

SimpleHub 支持 5 种 API 聚合站点类型，每种类型的认证方式、请求路径、数据格式各不相同。本文档记录每种类型对应的代码位置和核心逻辑。

## 类型概览

| 类型 | 认证方式 | 用户标头 | 模型端点 | 签到 | 账单 | tokens 路径 |
|------|----------|----------|----------|------|------|-------------|
| [newapi](newapi.md) | Bearer | `New-Api-User` | `/api/user/models` | ✅ `/api/user/checkin` | `/api/user/self` (÷500000) | `/api/token/` |
| [veloera](veloera.md) | Bearer | `Veloera-User` | `/api/user/models` | ✅ `/api/user/check_in` | `/api/user/self` (÷500000) | `/api/token/` |
| [voapi](voapi.md) | 原始 Key | 无 | `/api/models` | ✅ `/api/check_in` | `/api/user/info` (原始值) | `/api/keys` |
| [donehub](donehub.md) | Bearer | 无 | `/api/available_model` | ❌ | `/api/user/self` (÷500000) | `/api/token/` |
| [other](other.md) | Bearer | 无 | `/v1/models` | ❌ | 自定义 / OpenAI Billing | 无 |

## 涉及 API 类型逻辑的核心文件

| 文件 | 作用 |
|------|------|
| `internal/handler/site_handler.go` | 令牌、分组、定价、兑换的 HTTP 代理路由和请求/响应转换 |
| `internal/checker/runner.go` | 模型获取、账单请求、签到请求、模型解析 |
| `internal/service/check_service.go` | 将 `site.APIType` 传递给检查引擎 |
| `internal/model/site.go` | Site 模型定义，默认 `apiType: "other"` |
| `internal/handler/export_handler.go` | 导入时默认值处理 |

## 代理路由核心逻辑 (`site_handler.go`)

每种 API 类型在 `doProxy` 函数中通过 `pathByType` 映射路径，通过 `authType` 控制认证方式，通过 `transformRequest` / `transform` 进行请求/响应转换。
