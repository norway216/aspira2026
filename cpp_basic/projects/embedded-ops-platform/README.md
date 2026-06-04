# Embedded Ops Platform

分布式嵌入式设备 Web 运维管理系统 — 基于 Go + Vue 3 + PostgreSQL + Redis 构建。

## Architecture

```
┌────────────────────────────────────────────┐
│            Frontend (Vue 3 + Naive UI)     │
│          Nginx (static files + proxy)      │
└──────────────────┬─────────────────────────┘
                   │ /api/* /ws
                   ▼
┌────────────────────────────────────────────┐
│         Backend (Go + Gin)                 │
│   Device / Agent / Command / Log / Alert   │
└──────────────┬────────────────┬────────────┘
               │                │
               ▼                ▼
       ┌────────────┐    ┌──────────┐
       │ PostgreSQL │    │  Redis   │
       │ (persist)  │    │ (online) │
       └────────────┘    └──────────┘
               ▲
               │ Agent / ws / http
               ▼
       ┌──────────────────┐
       │  Agent (Go)      │
       │  RK3568/RK3588/  │
       │  Jetson/ARM      │
       └──────────────────┘
```

## Quick Start

### Docker 部署

```bash
# 一键部署（构建 + 启动）
./deploy.sh

# 仅构建镜像
./deploy.sh --build

# 仅启动
./deploy.sh --start

# 查看状态
./deploy.sh --status

# 查看日志
./deploy.sh --logs

# 重启
./deploy.sh --restart

# 停止
./deploy.sh --stop

# 清理所有数据
./deploy.sh --clean
```

### 本地开发（无需 Docker）

```bash
# 编译后端和 Agent
./deploy.sh --build-local

# 启动后端（需要本地 PostgreSQL + Redis）
./deploy.sh --run-local

# 启动 Agent
./deploy.sh --run-agent

# 开发前端
cd frontend && npm install && npm run dev
```

### Agent 部署到嵌入式设备

```bash
# 交叉编译 ARM64
./deploy.sh --build-agent-arm

# 创建部署包
./deploy.sh --agent-package

# 在目标设备上
tar xzf agent-package.tar.gz
cd agent-package && sudo ./install.sh
```

部署后访问:
- **前端**: http://localhost:80
- **后端**: http://localhost:8080/health
- **默认登录**: `admin` / `admin123`

## Key Features

- **零接触注册**: Agent 携带一次性注册码自动注册
- **高并发**: Redis 缓存在线状态，批量写入指标
- **实时通信**: WebSocket 推送设备状态和指标更新
- **安全设计**: JWT 用户认证 + HMAC Agent Token + 命令白名单
- **嵌入式专项检测**: GPU/Mali/DRI, USB, WiFi, 音频, 温度

## 支持的嵌入式平台

- RK3568 / RK3576 / RK3588
- NVIDIA Jetson (Orin/TX2/Xavier)
- ARM Debian/Ubuntu
- 自研 BSP 设备

## 命令白名单

| Action | 说明 | 风险 |
|--------|------|------|
| `get_basic_info` | 系统基础信息 | 低 |
| `get_dmesg` | 内核日志 | 低 |
| `get_usb_devices` | USB 设备列表 | 低 |
| `get_audio_status` | 声卡状态 | 低 |
| `get_gpu_status` | GPU/DRI/Mali 状态 | 低 |
| `get_wifi_status` | WiFi 模块状态 | 低 |
| `set_audio_volume` | 设置音量 | 中 |
| `restart_app` | 重启指定服务 | 中 |
| `upload_log_bundle` | 上传日志包 | 中 |
| `reboot_device` | 重启设备 | 高(默认禁用) |

## Agent Configuration

Agent 配置文件: `/etc/embedded-agent/config.yaml`

```yaml
server:
  url: "http://your-server:8080"
  websocket_url: "ws://your-server:8080/ws"

agent:
  register_code: "embedded-ops-first-run"

collector:
  heartbeat_interval_sec: 5
  metrics_interval_sec: 10

command:
  enable_remote_command: true
  allow_reboot: false
```

## Environment Variables

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_USER` | `embedded_ops` | 数据库用户 |
| `DB_PASSWORD` | `embedded_ops_secret` | 数据库密码 |
| `DB_NAME` | `embedded_ops` | 数据库名 |
| `SERVER_PORT` | `8080` | 后端端口 |
| `FRONTEND_PORT` | `80` | 前端端口 |
| `JWT_SECRET` | 自动生成 | JWT 密钥 |
| `GIN_MODE` | `release` | Gin 模式 |

## Tech Stack

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3 + TypeScript + Naive UI + ECharts + Pinia |
| 后端 | Go + Gin + WebSocket + JWT |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| Agent | Go (静态编译) + systemd |
| 部署 | Docker Compose |

## License

MIT
