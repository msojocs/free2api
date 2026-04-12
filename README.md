# Auto Register — 账号管理系统

> [English](README_EN.md)

基于 **Golang (Gin)** + **React 19 (Ant Design)** 构建的通用账号批量注册与管理系统。

## 架构

```
核心引擎 + 资源池 + 平台插件
```

| 层级 | 技术栈 |
|------|--------|
| Web 界面 | React 19 + Ant Design 5 |
| API 服务 | Go 1.21 + Gin |
| 任务引擎 | 协程池 + Channel 任务队列 |
| 资源池 | 代理 IP、邮箱、验证码 Token |
| 数据存储 | GORM + SQLite（可迁移至 PostgreSQL）|

## 快速开始

### Docker Compose 方式

```bash
cp .env.example .env        # 设置 JWT_SECRET 和 ENCRYPTION_KEY
docker compose up -d
```

- 前端：http://localhost:3000  
- API：http://localhost:8080

### 本地开发

**后端：**
```bash
cd server
go mod download
JWT_SECRET=your-secret go run ./cmd/
```

首次启动时，若数据库为空，服务会自动使用 [server/config/config.yaml](server/config/config.yaml) 中的配置或环境变量 `DEFAULT_ADMIN_USERNAME` / `DEFAULT_ADMIN_PASSWORD` 创建默认管理员账号（默认值：`admin` / `admin123456`）。

SQLite 使用 `github.com/mattn/go-sqlite3`，该驱动需要启用 CGO。在 Windows 本地运行时，需先安装 C 工具链并确保 `CGO_ENABLED=1`。

**前端：**
```bash
cd web
npm install
npm run dev
```

## 功能特性

- **任务调度** — 生产者-消费者模型；每次运行生成唯一 BatchID；支持单账号维度追踪（IP、时间、错误堆栈）
- **代理管理** — 静态 IP 与隧道代理；403/429 时自动加黑名单
- **邮件中心** — IMAP 自动收信；正则提取验证码；一对一绑定
- **验证码处理** — YesCaptcha / 2Captcha / CapSolver；自动切换；费用监控
- **自动化调度** — Cron 任务自动补货与 Session 健康检查
- **实时进度** — 每任务 SSE 推流；React 19 Dashboard
- **安全机制** — AES-CFB 加密密码/Token；JWT 鉴权；QPS 限流

## 目录结构

```
├── server/                 # Go 后端
│   ├── cmd/                # 入口
│   ├── internal/
│   │   ├── api/            # Gin 路由与处理器
│   │   ├── model/          # GORM 模型
│   │   ├── service/        # 业务逻辑
│   │   ├── core/           # 协程池
│   │   ├── executor/       # 平台插件（chatgpt、cursor 等）
│   │   ├── resource/       # 代理 / 邮件 / 验证码管理
│   │   └── scheduler/      # Cron 任务
│   └── pkg/                # 加密、浏览器指纹、HTTP 工具
├── web/                    # React 前端
│   └── src/
│       ├── api/            # Axios API 模块
│       ├── components/     # AppLayout、StatusTag、ResourceCard、TaskProgress
│       ├── pages/          # Dashboard、TaskList、AccountList、ProxyManager、MailManager
│       └── store/          # Zustand 认证 Store
└── docker-compose.yml
```

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `JWT_SECRET` | *(必填)* | JWT 签名密钥 |
| `ENCRYPTION_KEY` | *(必填)* | 16/24/32 字节 AES 密钥 |
| `GIN_MODE` | `debug` | 生产环境设为 `release` |
| `PORT` | `8080` | 服务监听端口 |
| `DB_DRIVER` | `sqlite` | 数据库驱动：`sqlite` 或 `postgres` |
| `DB_HOST` | `127.0.0.1` | PostgreSQL 主机地址 |
| `DB_PORT` | `5432` | PostgreSQL 端口 |
| `DB_NAME` | — | 数据库名称 |
| `DB_USER` | — | 数据库用户 |
| `DB_PASSWORD` | — | 数据库密码 |
| `DB_SSL_MODE` | `disable` | PostgreSQL SSL 模式 |
| `DB_TIMEZONE` | `Local` | 数据库时区 |

## API 概览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 登录，返回 JWT Token |
| POST | `/api/auth/register` | 注册管理员 |
| GET | `/api/dashboard/stats` | 仪表盘统计数据 |
| GET/POST | `/api/tasks` | 任务列表 / 创建任务 |
| POST | `/api/tasks/:id/start` | 启动任务 |
| POST | `/api/tasks/:id/pause` | 暂停任务 |
| GET | `/api/tasks/:id/progress` | SSE 实时进度流 |
| GET | `/api/accounts` | 账号列表（支持平台/状态筛选）|
| GET | `/api/accounts/export` | 导出 CSV 或 JSON |
| GET/POST | `/api/proxies` | 代理列表 / 添加代理 |
| POST | `/api/proxies/:id/test` | 测试代理连通性 |
| GET/POST | `/api/mails` | 邮箱列表 / 添加邮箱 |
| GET | `/api/captcha/stats` | 验证码费用统计 |

## 感谢

感谢以下项目提供的思路：

https://github.com/zc-zhangchen/any-auto-register

## 许可证

本项目采用 [MIT License](LICENSE)。
