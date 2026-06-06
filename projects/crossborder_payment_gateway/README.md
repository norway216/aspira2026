# Aspira Cross-Border Payment Gateway

高性能、稳定的跨境支付网关系统，支持实时交易处理、分布式部署、审计追踪和Web管理面板。

## 系统架构

```
用户客户端 -> Go API网关 -> C++ 交易引擎 -> 数据库
                |              |
             WebSocket      消息队列
                |              |
            Web管理面板    审计日志系统
```

## 技术栈

| 模块 | 技术 | 说明 |
|------|------|------|
| API网关 | Go (Gin) | HTTP/HTTPS, JWT认证, WebSocket实时推送 |
| 交易引擎 | C++20 | 无锁队列, 工作窃取线程池, SHA-256哈希链审计 |
| 数据库 | SQLite / PostgreSQL | 开发环境SQLite, 生产环境PostgreSQL |
| 缓存 | Redis | 会话缓存, 速率限制 |
| 前端 | Vanilla JS + ECharts | Apple Music风格深色主题, SPA |
| 容器化 | Docker + Docker Compose | 一键部署 |

## 项目结构

```
crossborder_payment_gateway/
├── gateway/          # Go API网关 (Gin框架)
│   ├── main.go       # 入口, 路由, 中间件
│   ├── config/       # 配置加载
│   └── internal/
│       ├── database/ # 数据库接口 + SQLite实现
│       ├── models/   # 数据模型
│       ├── auth/     # JWT + bcrypt认证
│       ├── handler/  # HTTP处理器
│       ├── middleware/# 中间件(CORS, 限流, 审计)
│       ├── engine/   # C++引擎客户端
│       └── websocket/# WebSocket实时推送
├── engine/           # C++高性能交易引擎
│   ├── src/pipeline/ # 无锁队列 + 线程池
│   ├── src/network/  # TCP服务器
│   ├── src/processor/# 交易处理, 验证, 汇率
│   └── src/crypto/   # AES加密 + 哈希链
├── dashboard/        # Web管理面板
│   ├── index.html    # SPA入口
│   └── static/
│       ├── css/      # Apple Music深色主题
│       └── js/       # SPA路由, 页面, 组件
├── client/           # 性能测试客户端
├── deploy/           # Docker部署配置
└── README.md
```

## 快速开始

### 方式一: 直接运行 (开发模式)

**前置要求:** Go 1.22+, GCC

```bash
# 1. 启动网关 (内置交易引擎)
cd gateway
go run . configs/config.yaml

# 2. 打开浏览器访问
# http://localhost:8080

# 默认账号: admin / admin123
```

### 方式二: Docker Compose

```bash
# 构建并启动所有服务
docker-compose -f deploy/docker-compose.yml up -d

# 查看日志
docker-compose -f deploy/docker-compose.yml logs -f gateway

# 停止
docker-compose -f deploy/docker-compose.yml down
```

### 方式三: 包含C++引擎的完整部署

```bash
# 1. 构建C++引擎
cd engine && mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release .. && make -j$(nproc)

# 2. 启动引擎
./payment_engine ../config/engine.json

# 3. 修改gateway配置启用引擎
# 编辑 gateway/configs/config.yaml:
#   engine.enabled: true

# 4. 启动网关
cd gateway && go run .
```

## API 端点

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 用户登录 |
| POST | `/api/v1/auth/refresh` | 刷新令牌 |
| GET | `/api/v1/auth/profile` | 获取用户信息 |

### 交易
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/transactions` | 创建交易 |
| GET | `/api/v1/transactions` | 查询交易列表 |
| GET | `/api/v1/transactions/:id` | 交易详情 |
| POST | `/api/v1/transactions/:id/refund` | 退款 |

### 仪表盘
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/dashboard` | 仪表盘统计 |
| GET | `/api/v1/dashboard/tps-history` | TPS历史数据 |
| GET | `/api/v1/dashboard/recent-transactions` | 最近交易 |

### WebSocket
| 端点 | 说明 |
|------|------|
| `ws://localhost:8080/ws` | 实时数据推送 |

## 性能测试

使用内置的基准测试客户端:

```bash
cd client

# 编译
go build -o client .

# 支付场景测试 (20并发, 限速1000 TPS, 持续30秒)
./client --mode=payment --concurrency=20 --rate=1000 --duration=30s

# 混合场景测试 (50并发, 不限速)
./client --mode=mixed --concurrency=50 --duration=30s

# 查询场景测试
./client --mode=query --concurrency=10 --duration=10s

# 保存报告
./client --mode=payment --concurrency=100 --rate=5000 --duration=60s --report=results.json
```

## Web管理面板功能

- **仪表盘**: 实时TPS统计卡, 交易量图表, 最近交易列表, 引擎健康状态
- **交易管理**: 交易列表查询, 状态筛选, 退款操作
- **账户管理**: 账户余额查看, 交易流水
- **商户管理**: 商户CRUD, API密钥管理
- **审计日志**: 操作日志查询, 可追溯
- **汇率管理**: 多币种汇率配置

## 核心特性

1. **高并发交易处理**: C++无锁队列 + 工作窃取线程池, Go协程并发处理
2. **审计哈希链**: SHA-256链接每笔交易, 不可篡改, 可验证
3. **实时监控**: WebSocket推送交易状态, TPS, 引擎健康
4. **故障容错**: C++引擎不可用时自动回退到内部Go引擎
5. **Apple Music风格UI**: 深色主题, 玻璃拟态效果, 渐变色卡片
6. **数据加密**: AES-256-GCM传输加密, bcrypt密码哈希

## License

Copyright © 2026 Aspira Studio. All rights reserved.
