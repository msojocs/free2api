# free2api — Account Management System

A universal account batch-registration and management system built with **Golang (Gin)** + **React 19 (Ant Design)**.

## Architecture

```
Core Engine + Resource Pool + Platform Plugins
```

| Layer | Technology |
|-------|-----------|
| Web UI | React 19 + Ant Design 5 |
| API Server | Go 1.21 + Gin |
| Worker Engine | Goroutine pool with channel-based task queue |
| Resource Pool | Proxy IP, Mailbox, Captcha Token |
| Storage | GORM + SQLite (migrates to MySQL/Postgres) |

## Quick Start

### With Docker Compose

```bash
cp .env.example .env        # set JWT_SECRET and ENCRYPTION_KEY
docker compose up -d
```

- Frontend: http://localhost:3000  
- API: http://localhost:8080

### Development

**Backend:**
```bash
cd server
go mod download
JWT_SECRET=your-secret go run ./cmd/
```

On first startup with an empty database, the server creates a default admin account using the values in [server/config/config.yaml](/home/msojocs/github/free2api/server/config/config.yaml) or the `DEFAULT_ADMIN_USERNAME` / `DEFAULT_ADMIN_PASSWORD` environment variables. The defaults are `admin` / `admin123456`.

SQLite uses github.com/mattn/go-sqlite3 in this project.
That driver creates the database file automatically when it opens successfully, but it requires CGO to be enabled.
If you run locally on Windows, install a C toolchain first and make sure `CGO_ENABLED=1`.

**Frontend:**
```bash
cd web
npm install
npm run dev
```

## Features

- **Task Scheduling** — Producer-consumer model; unique BatchID per run; per-account trace (IP, time, error stack)
- **Proxy Manager** — Static IP & tunnel proxy; auto-blacklist on 403/429
- **Mail Center** — IMAP auto-retrieval; regex-based code extraction; one-to-one binding
- **Captcha Handler** — YesCaptcha / 2Captcha / CapSolver; auto-failover; cost monitoring
- **Automation** — Cron jobs for inventory replenishment and session health checks
- **Real-time Progress** — SSE stream per task; React 19 Dashboard
- **Security** — AES-CFB encrypted passwords/tokens; JWT auth; QPS limiting

## Directory Structure

```
├── server/                 # Go backend
│   ├── cmd/                # Entry point
│   ├── internal/
│   │   ├── api/            # Gin Router & Handlers
│   │   ├── model/          # GORM Models
│   │   ├── service/        # Business logic
│   │   ├── core/           # Worker Pool
│   │   ├── executor/       # Platform plugins (chatgpt, cursor)
│   │   ├── resource/       # Proxy / Mail / Captcha managers
│   │   └── scheduler/      # Cron jobs
│   └── pkg/                # Crypto, browser fingerprint, HTTP util
├── web/                    # React frontend
│   └── src/
│       ├── api/            # Axios API modules
│       ├── components/     # AppLayout, StatusTag, ResourceCard, TaskProgress
│       ├── pages/          # Dashboard, TaskList, AccountList, ProxyManager, MailManager
│       └── store/          # Zustand auth store
└── docker-compose.yml
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(required)* | JWT signing secret |
| `ENCRYPTION_KEY` | *(required)* | 16/24/32-byte AES key |
| `GIN_MODE` | `debug` | `release` in production |
| `PORT` | `8080` | Server port |

## API Overview

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/login` | Login → JWT token |
| POST | `/api/auth/register` | Register admin |
| GET | `/api/dashboard/stats` | Dashboard statistics |
| GET/POST | `/api/tasks` | List / create task batches |
| POST | `/api/tasks/:id/start` | Start a task |
| POST | `/api/tasks/:id/pause` | Pause a task |
| GET | `/api/tasks/:id/progress` | SSE progress stream |
| GET | `/api/accounts` | List accounts (filter by platform/status) |
| GET | `/api/accounts/export` | Export CSV or JSON |
| GET/POST | `/api/proxies` | List / add proxies |
| POST | `/api/proxies/:id/test` | Test proxy connectivity |
| GET/POST | `/api/mails` | List / add mailboxes |
| GET | `/api/captcha/stats` | Captcha cost statistics |
