# SimpleHub · Go

SimpleHub 的 Go 后端迁移版本 — 统一管理 NewAPI / Veloera / VOAPI / DoneHub 等 AI API 聚合站点。

## 功能

- 多站点统一管理（NewAPI / Veloera / VOAPI / DoneHub / Other）
- 模型列表拉取、差异变更追踪
- 自动/手动签到（Check-in）
- 额度查询
- 定时批量检测 + 邮件通知（Resend）
- 令牌管理、分组查询、定价查询（代理到上游 API）
- 站点导出/导入
- 单二进制部署，前端内嵌

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端框架 | Gin |
| ORM | GORM |
| 数据库 | SQLite (modernc.org/sqlite, 纯 Go, 无需 CGO) |
| 认证 | golang-jwt v5 |
| 日志 | zerolog |
| 邮件 | Resend SDK |
| 定时 | robfig/cron v3 |
| 前端 | React 18 + Ant Design 5 + Vite |
| 打包 | //go:embed 内嵌前端 dist |

## 快速开始

### 前置要求

- Go 1.22+
- Node.js 18+（构建前端时需要）

### Windows

```powershell
.\build.ps1
.\bin\server.exe
```

### Linux / macOS

```bash
make build
./bin/server
```

### Docker

```bash
make docker-build
make docker-run
```

### 仅构建后端（使用占位前端）

```bash
make build-go
make run
```

首次运行会自动生成端口、安全入口路径、管理员账号密码，打印在控制台。**请立即记录这些信息。**

## 重置

```bash
# Windows
.\bin\server.exe reset

# Linux / macOS
./bin/server reset
```

重置会重新生成端口、安全入口、管理员账号密码、JWT 密钥、加密密钥。已有站点数据不受影响。

## 安全入口

系统启动时会生成一个随机 8 位十六进制路径（如 `a3f1b2c8`）。登录页仅能通过此路径访问：

```
http://localhost:PORT/ENTRY
```

直接访问 `/` 会提示"请通过安全入口登录"。

## 项目结构

```
├── cmd/server/       # 入口：main.go + 内嵌前端
│   └── dist/         # 前端构建产物占位
├── internal/
│   ├── checker/      # 检测引擎
│   ├── config/       # 运行时配置
│   ├── crypto/       # AES 加解密
│   ├── handler/      # HTTP handlers
│   ├── middleware/    # Gin 中间件（认证/CORS/日志）
│   ├── model/        # GORM 模型
│   ├── proxy/        # 上游 API 代理客户端
│   ├── repository/   # 数据库访问层
│   ├── router/       # 路由注册
│   └── service/      # 业务逻辑层
├── web/              # 前端源码（React + Ant Design + Vite）
├── scripts/          # 构建脚本 + 测试脚本
├── docs/             # API 文档
├── build.ps1         # Windows 构建
├── build.sh          # Linux/macOS 构建
├── Makefile          # 构建任务
└── Dockerfile        # 多阶段 Docker 构建
```

## API 文档

详见 [`docs/api.md`](docs/api.md) 或原始规范 [`docs/api.jsonc`](docs/api.jsonc)。

## 测试

```bash
python scripts/test_api.py
```

包含 25 个 E2E 测试用例，覆盖认证、站点 CRUD、检测、代理、导出导入、分类、邮件和计划任务配置。

## 许可证

MIT

## 致谢

本项目基于 [SimpleHub](https://github.com/jwy87/SimpleHub)（Node.js 版本）迁移至 Go，感谢原作者的工作。
