# Secure Gateway - 分布式安全流量网关

基于 Go + Vue 3 的分布式安全流量网关系统，支持多节点部署、加密通信、负载均衡、高并发连接处理、用户管理、节点监控和流量统计。

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend   │    │   API Gateway    │    │   Traffic Node   │
│   Vue 3 + TS     │◄──►│   Go + Gin       │◄──►│   Go Proxy       │
│   Element Plus   │    │   JWT + RBAC     │    │   TLS + Metrics  │
└─────────────────┘    └───────┬──────────┘    └─────────────────┘
                               │
                        ┌──────┴──────┐
                        │  PostgreSQL  │
                        │  + Redis     │
                        └─────────────┘
```

## 快速开始

### 前置条件

- Docker & Docker Compose
- Node.js 18+
- 或 Go 1.22+ (本地开发)

### 一键启动（推荐）

```bash
# 给予执行权限
chmod +x scripts/dev.sh

# 启动所有服务
./scripts/dev.sh
```

### 手动启动

```bash
# 1. 先启动数据库
docker compose up -d postgres redis

# 2. 构建并启动 API Gateway + Traffic Node
docker compose up -d api-gateway traffic-node

# 3. 构建前端并启动 Web
cd web
npm install
npm run build
cd ..
docker compose up -d web
```

### 访问系统

| 服务 | 地址 | 说明 |
|------|------|------|
| Web UI | http://localhost:80 | 管理后台 |
| API | http://localhost:8080 | REST API |
| Metrics | http://localhost:9100/metrics | Prometheus 指标 |

**默认登录账号:** `admin` / `admin123`

## API 接口

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 用户登录 |
| POST | /api/v1/auth/refresh | 刷新 Token |
| POST | /api/v1/auth/logout | 登出 |
| GET | /api/v1/auth/profile | 获取当前用户信息 |

### 仪表盘

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/dashboard/stats | 系统概览统计 |
| GET | /api/v1/dashboard/distribution | 节点区域分布 |
| GET | /api/v1/dashboard/traffic-history | 流量历史曲线 |
| GET | /api/v1/dashboard/top-users | 流量用户排行 |

### 用户管理（管理员）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/users | 用户列表 |
| POST | /api/v1/users | 创建用户 |
| PUT | /api/v1/users/:id | 更新用户 |
| DELETE | /api/v1/users/:id | 删除用户 |

### 节点管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/nodes | 节点列表 |
| POST | /api/v1/nodes/register | 节点注册 |
| POST | /api/v1/nodes/heartbeat | 节点心跳 |
| GET | /api/v1/nodes/:node_id/metrics | 节点指标 |

### 流量统计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/traffic | 流量记录 |
| GET | /api/v1/traffic/summary | 流量汇总 |

## 项目结构

```
secure-gateway/
├── cmd/                    # Go 入口
│   ├── api-gateway/        # API 网关服务
│   └── traffic-node/       # 流量节点服务
├── internal/               # 内部包
│   ├── auth/               # JWT 认证
│   ├── config/             # 配置管理
│   ├── database/           # 数据库操作
│   ├── handler/            # HTTP 处理器
│   ├── middleware/          # Gin 中间件
│   ├── metrics/            # 流量统计
│   ├── model/              # 数据模型
│   └── proxy/              # TCP 代理
├── pkg/                    # 公共包
│   ├── logger/             # 日志
│   └── limiter/            # 限流器
├── web/                    # Vue 3 前端
├── configs/                # YAML 配置
├── scripts/                # 工具脚本
├── deploy/                 # 部署配置
├── docker-compose.yml
├── Dockerfile.api-gateway
├── Dockerfile.traffic-node
└── README.md
```

## 本地开发（无 Docker）

如果需要本地运行 Go 服务而不使用 Docker：

```bash
# 安装依赖
go mod tidy

# 运行 API Gateway
go run ./cmd/api-gateway -config configs/api-gateway.yaml

# 运行 Traffic Node（新终端）
go run ./cmd/traffic-node -config configs/traffic-node.yaml
```

## 技术栈

- **后端语言:** Go 1.22
- **Web 框架:** Gin
- **数据库:** PostgreSQL 16 + Redis 7
- **前端:** Vue 3 + TypeScript + Element Plus + ECharts
- **认证:** JWT (Access Token + Refresh Token)
- **部署:** Docker Compose
- **监控:** Prometheus Metrics

## 安全注意事项

1. 生产环境必须启用 TLS
2. JWT 签名密钥通过环境变量设置，不要硬编码
3. 数据库密码使用强密码
4. 默认管理员密码 `admin123` 仅用于开发，生产环境请修改

## License

MIT