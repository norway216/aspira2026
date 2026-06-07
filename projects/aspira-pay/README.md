# Aspira Pay V2 — 跨境支付清算交易系统

> 版本：V2 Sandbox / 准生产级架构
> 技术栈：Go + C++20 + PostgreSQL + NATS + Redis + Blockchain

## 系统概述

Aspira Pay V2 是一个面向跨境支付、清算、审计和链上追溯的交易系统。

核心架构：
- **Go** — 支付业务编排、KYC、风控、订单状态机、清算服务
- **C++20** — 高性能交易清算引擎（资金冻结、扣减、入账、WAL）
- **PostgreSQL** — 业务账本、复式记账
- **NATS JetStream** — 事件驱动消息队列
- **Redis** — 缓存、限流
- **区块链（Hash Chain / Merkle Tree）** — 交易审计不可篡改证明

## 项目结构

```
aspira-pay/
├── backend-go/          # Go API 单体（合并所有业务模块）
├── engine-cpp/          # C++ 交易清算引擎
├── web-admin/           # React 管理后台
├── migrations/          # PostgreSQL 数据库迁移
├── deploy/              # Docker Compose / K8s 部署配置
└── README.md
```

## 快速开始

### 前置依赖

- Go 1.22+
- CMake 3.16+ & C++20 编译器
- Docker & Docker Compose
- PostgreSQL 16+
- Redis 7+

### 最小部署（Docker Compose）

```bash
cd deploy
docker-compose up -d
```

服务端口：
- API Gateway: `http://localhost:8080`
- Web Admin: `http://localhost:3000`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001`

### 本地开发

```bash
# 1. 启动基础设施
cd deploy && docker-compose up -d postgres redis nats

# 2. 运行数据库迁移
psql -h localhost -U aspirapay -d aspirapay -f migrations/001_init_users.sql
# ... 依次执行所有迁移文件

# 3. 构建 C++ 引擎
cd engine-cpp && mkdir build && cd build && cmake .. && make -j4

# 4. 启动 Go API
cd backend-go && go run cmd/server/main.go -config configs/config.yaml
```

## API 基础路径

```
Base URL: http://localhost:8080/api/v2
```

## 关键技术原则

1. 金额全部使用 int64 最小货币单位
2. 所有交易接口必须幂等
3. 账本只能追加，不能删除
4. 所有交易必须有状态机
5. C++ 引擎只负责高性能执行，Go 负责业务编排
6. 本地账本完成优先，链上确认最终一致

## License

Proprietary — Aspira Studio
