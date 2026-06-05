# 小型多节点代理/流量调度实验平台架构设计文档

> 项目定位：自用、实验室验证、网络流量调度与负载控制学习项目  
> 推荐项目名：**TrafficLab Proxy Control Platform**  
> 英文项目名：**Multi-node Traffic Scheduling and Proxy Control Platform**

---

## 1. 项目目标

本项目不是面向商业化运营的“机场”系统，而是一个用于学习和验证网络系统设计的实验平台。核心目标是通过多节点代理转发、流量统计、限速控制、负载调度和监控可视化，掌握高性能网络服务的工程实现方法。

### 1.1 核心学习目标

- 理解 TCP/UDP 流量转发原理
- 掌握多节点代理架构设计
- 实现用户级、节点级流量统计
- 实现限速策略，例如 Token Bucket
- 实现节点负载均衡策略
- 实现节点心跳、健康检查和故障摘除
- 使用 Prometheus + Grafana 做监控可视化
- 使用 Docker Compose 搭建实验环境
- 为后续学习高性能 C++ 网络编程、分布式网关、边缘网络服务打基础

### 1.2 不做的内容

本项目不研究、不实现以下内容：

- 绕过网络监管或封锁的对抗性技术
- 流量伪装、探测规避、审查对抗
- 大规模商业化售卖系统
- 非法用途的匿名访问或滥用服务
- 绕过第三方服务条款的访问方式

建议把项目定位为：

> 私有代理网络实验平台 / 多节点流量调度实验系统 / 企业内网代理网关原型

---

## 2. 总体架构

### 2.1 总体拓扑

```text
                    ┌─────────────────────────────┐
                    │      Control Center          │
                    │ 控制中心 / 管理后台 / API服务 │
                    └──────────────┬──────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
              ↓                    ↓                    ↓
      ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
      │ Proxy Node A │     │ Proxy Node B │     │ Proxy Node C │
      │ 流量转发      │     │ 流量转发      │     │ 流量转发      │
      │ 用户限速      │     │ 用户限速      │     │ 用户限速      │
      │ 流量统计      │     │ 流量统计      │     │ 流量统计      │
      └──────┬───────┘     └──────┬───────┘     └──────┬───────┘
             │                    │                    │
             └────────────┬───────┴────────────┬───────┘
                          ↓                    ↓
                  Target Service         External/Internal Network
```

### 2.2 系统角色

| 角色 | 说明 |
|---|---|
| Client | 实验客户端，产生测试流量 |
| Control Center | 控制中心，负责用户、节点、策略、监控和调度 |
| Proxy Node | 代理节点，负责接收连接、转发流量、统计流量、执行限速 |
| Target Server | 目标服务，可以是 iperf3、nginx、HTTP server 或自建测试服务 |
| Monitoring Stack | Prometheus、Grafana、Node Exporter、日志系统 |

---

## 3. 核心模块设计

## 3.1 控制中心 Control Center

控制中心是整个系统的大脑，负责管理用户、节点、策略和监控数据。

### 3.1.1 功能列表

| 模块 | 功能 |
|---|---|
| 用户管理 | 创建用户、禁用用户、设置流量额度、设置限速 |
| 节点管理 | 注册节点、维护节点状态、查看节点负载 |
| 策略管理 | 配置用户限速、节点限速、调度策略 |
| 调度中心 | 根据节点状态选择合适节点 |
| 流量统计 | 接收节点上报的用户流量和节点流量 |
| 心跳检测 | 判断节点在线、离线、异常、拥塞 |
| 监控接口 | 提供 Prometheus metrics |
| 管理后台 | 展示节点状态、用户流量、系统事件 |

### 3.1.2 推荐技术栈

| 组件 | 推荐方案 |
|---|---|
| 后端语言 | Python FastAPI / Go |
| 数据库 | PostgreSQL / SQLite |
| 缓存 | Redis，可选 |
| 前端 | Vue / React，可选 |
| 部署 | Docker Compose |
| 监控 | Prometheus + Grafana |
| 日志 | Loki / 本地文件日志 |

### 3.1.3 控制中心 API 设计

#### 节点注册接口

```http
POST /api/v1/nodes/register
```

请求示例：

```json
{
  "node_id": "node-a",
  "name": "Proxy Node A",
  "ip": "192.168.1.101",
  "proxy_port": 1080,
  "max_bandwidth_mbps": 100,
  "max_connections": 1000,
  "node_secret": "node-secret-token"
}
```

响应示例：

```json
{
  "success": true,
  "node_id": "node-a",
  "message": "node registered"
}
```

#### 节点心跳接口

```http
POST /api/v1/nodes/heartbeat
```

请求示例：

```json
{
  "node_id": "node-a",
  "cpu_usage": 35.2,
  "memory_usage": 48.7,
  "current_connections": 128,
  "rx_mbps": 32.5,
  "tx_mbps": 28.1,
  "status": "online",
  "timestamp": "2026-06-04T10:00:00Z"
}
```

#### 节点流量上报接口

```http
POST /api/v1/traffic/report
```

请求示例：

```json
{
  "node_id": "node-a",
  "records": [
    {
      "user_id": "user001",
      "upload_bytes": 104857600,
      "download_bytes": 524288000,
      "connection_count": 3
    }
  ],
  "timestamp": "2026-06-04T10:00:00Z"
}
```

#### 策略拉取接口

```http
GET /api/v1/nodes/{node_id}/policies
```

响应示例：

```json
{
  "node_id": "node-a",
  "global_policy": {
    "max_node_bandwidth_mbps": 100,
    "max_node_connections": 1000
  },
  "user_policies": [
    {
      "user_id": "user001",
      "max_rate_mbps": 10,
      "burst_mbps": 30,
      "max_connections": 5,
      "status": "active"
    },
    {
      "user_id": "user002",
      "max_rate_mbps": 20,
      "burst_mbps": 50,
      "max_connections": 10,
      "status": "active"
    }
  ]
}
```

---

## 3.2 代理节点 Proxy Node

代理节点是数据平面，负责实际流量转发和策略执行。

### 3.2.1 功能列表

| 模块 | 功能 |
|---|---|
| Listener | 监听客户端连接 |
| Authenticator | 验证用户 token 或账号 |
| TCP Proxy | 建立客户端到目标服务的 TCP 转发 |
| UDP Proxy | 可作为二期功能 |
| Traffic Counter | 统计用户上传和下载字节数 |
| Rate Limiter | 执行限速策略 |
| Policy Manager | 从控制中心拉取策略 |
| Heartbeat Reporter | 定时上报节点状态 |
| Metrics Exporter | 暴露 Prometheus 指标 |
| Health Checker | 检查本机服务状态 |

### 3.2.2 代理节点内部结构

```text
┌────────────────────────────────────────────┐
│              Proxy Node                    │
│                                            │
│  ┌─────────────┐      ┌─────────────────┐  │
│  │ Listener    │ ---> │ Authenticator   │  │
│  └─────────────┘      └─────────────────┘  │
│          │                         │        │
│          ↓                         ↓        │
│  ┌──────────────────────────────────────┐  │
│  │          Connection Manager           │  │
│  └──────────────────────────────────────┘  │
│          │                         │        │
│          ↓                         ↓        │
│  ┌─────────────┐      ┌─────────────────┐  │
│  │ TCP Proxy   │ ---> │ Rate Limiter    │  │
│  └─────────────┘      └─────────────────┘  │
│          │                         │        │
│          ↓                         ↓        │
│  ┌─────────────┐      ┌─────────────────┐  │
│  │ Counter     │ ---> │ Reporter        │  │
│  └─────────────┘      └─────────────────┘  │
└────────────────────────────────────────────┘
```

### 3.2.3 推荐实现方式

第一版建议使用 C++20 实现代理节点，原因是可以训练网络编程和高性能数据转发能力。

推荐技术：

| 功能 | 推荐技术 |
|---|---|
| 网络 IO | epoll / asio |
| 并发模型 | 事件循环 + 线程池 |
| 配置文件 | YAML / JSON |
| HTTP 上报 | libcurl / Boost.Beast |
| 日志 | spdlog |
| 指标导出 | prometheus-cpp |
| 构建 | CMake |

---

## 4. 数据流设计

## 4.1 客户端访问数据流

```text
Client
  │
  │ 1. 连接 Proxy Node
  ↓
Proxy Node
  │
  │ 2. 验证用户身份
  ↓
Policy Manager
  │
  │ 3. 检查用户状态、连接数、限速策略
  ↓
TCP Proxy
  │
  │ 4. 建立到 Target Server 的连接
  ↓
Target Server
```

## 4.2 流量统计数据流

```text
Proxy Node
  │
  │ 1. 每条连接记录 upload/download bytes
  ↓
Traffic Counter
  │
  │ 2. 按用户聚合
  ↓
Traffic Reporter
  │
  │ 3. 每 10~60 秒上报控制中心
  ↓
Control Center
  │
  │ 4. 写入数据库
  ↓
Dashboard / Grafana
```

## 4.3 策略下发数据流

```text
Control Center
  │
  │ 1. 管理员修改策略
  ↓
Database
  │
  │ 2. 保存用户限速、节点限速
  ↓
Proxy Node
  │
  │ 3. 定时拉取最新策略
  ↓
Rate Limiter
  │
  │ 4. 应用新策略
```

---

## 5. 数据库设计

## 5.1 users 用户表

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    token VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    traffic_total BIGINT NOT NULL DEFAULT 0,
    traffic_used BIGINT NOT NULL DEFAULT 0,
    max_rate_mbps INTEGER NOT NULL DEFAULT 10,
    max_connections INTEGER NOT NULL DEFAULT 5,
    expired_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 5.2 nodes 节点表

```sql
CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    ip VARCHAR(64) NOT NULL,
    proxy_port INTEGER NOT NULL,
    max_bandwidth_mbps INTEGER NOT NULL DEFAULT 100,
    max_connections INTEGER NOT NULL DEFAULT 1000,
    status VARCHAR(32) NOT NULL DEFAULT 'offline',
    last_heartbeat TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 5.3 node_status_logs 节点状态日志表

```sql
CREATE TABLE node_status_logs (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL,
    cpu_usage DOUBLE PRECISION NOT NULL,
    memory_usage DOUBLE PRECISION NOT NULL,
    current_connections INTEGER NOT NULL,
    rx_mbps DOUBLE PRECISION NOT NULL,
    tx_mbps DOUBLE PRECISION NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 5.4 traffic_logs 用户流量日志表

```sql
CREATE TABLE traffic_logs (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    upload_bytes BIGINT NOT NULL DEFAULT 0,
    download_bytes BIGINT NOT NULL DEFAULT 0,
    connection_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 5.5 policies 策略表

```sql
CREATE TABLE policies (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    max_rate_mbps INTEGER NOT NULL,
    burst_mbps INTEGER NOT NULL,
    max_connections INTEGER NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 5.6 scheduler_events 调度事件表

```sql
CREATE TABLE scheduler_events (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(64),
    selected_node_id VARCHAR(64),
    strategy VARCHAR(64) NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 6. 限速策略设计

## 6.1 用户级限速

用户级限速控制单个用户的最大带宽。

例如：

```text
user001:
  max_rate = 10 Mbps
  burst = 30 Mbps
  max_connections = 5
```

效果：

- 长期平均速度不超过 10 Mbps
- 短时间允许突发到 30 Mbps
- 同时最多 5 条连接

## 6.2 节点级限速

节点级限速控制整个节点的出口能力。

例如：

```text
node-a:
  max_bandwidth = 100 Mbps
  max_connections = 1000
```

当节点达到阈值后：

```text
CPU > 85%
Memory > 90%
Bandwidth > 90%
Connections > 90%
```

节点状态可以标记为：

```text
degraded
```

控制中心减少对该节点的新连接分配。

## 6.3 Token Bucket 算法

Token Bucket 适合做带宽控制，因为它允许短时间突发，但长期速率可控。

### 6.3.1 工作原理

```text
1. 系统按固定速率生成 token
2. 每发送一定字节的数据，就消耗对应数量 token
3. token 足够，允许发送
4. token 不够，等待或丢弃
5. token 桶容量有限，因此只能允许有限突发
```

### 6.3.2 C++ 伪代码

```cpp
#include <chrono>
#include <algorithm>

class TokenBucket {
public:
    TokenBucket(double rate_bytes_per_sec, double burst_bytes)
        : rate_(rate_bytes_per_sec),
          capacity_(burst_bytes),
          tokens_(burst_bytes),
          last_refill_(std::chrono::steady_clock::now()) {}

    bool consume(size_t bytes) {
        refill();

        if (tokens_ >= static_cast<double>(bytes)) {
            tokens_ -= static_cast<double>(bytes);
            return true;
        }

        return false;
    }

private:
    void refill() {
        auto now = std::chrono::steady_clock::now();
        double elapsed = std::chrono::duration<double>(now - last_refill_).count();

        tokens_ = std::min(capacity_, tokens_ + elapsed * rate_);
        last_refill_ = now;
    }

private:
    double rate_;
    double capacity_;
    double tokens_;
    std::chrono::steady_clock::time_point last_refill_;
};
```

### 6.3.3 限速处理方式

当 token 不足时有两种方式：

| 方式 | 说明 |
|---|---|
| sleep 等待 | 更适合 TCP，降低发送速度 |
| 直接丢弃 | 更适合 UDP 或压力测试场景 |

对于 TCP 代理，建议使用 sleep 或事件循环延迟发送。

---

## 7. 负载调度策略设计

## 7.1 Round Robin 轮询

最简单的策略，按顺序选择节点。

```text
user1 -> node-a
user2 -> node-b
user3 -> node-c
user4 -> node-a
```

优点：

- 实现简单
- 适合节点性能接近的场景

缺点：

- 不考虑节点真实负载

## 7.2 Least Connections 最少连接数

选择当前连接数最少的节点。

```text
node-a: 100 connections
node-b: 40 connections
node-c: 80 connections

new connection -> node-b
```

优点：

- 适合长连接服务
- 比轮询更合理

缺点：

- 不考虑连接流量大小

## 7.3 Least Bandwidth 最低带宽占用

选择当前带宽占用最低的节点。

```text
node-a: 80 Mbps
node-b: 20 Mbps
node-c: 50 Mbps

new connection -> node-b
```

优点：

- 适合流量型服务
- 对带宽资源更敏感

缺点：

- 不考虑 CPU 和连接数

## 7.4 Weighted Round Robin 加权轮询

根据节点性能设置权重。

```text
node-a: weight = 5
node-b: weight = 3
node-c: weight = 1
```

理论分配比例：

```text
node-a : node-b : node-c = 5 : 3 : 1
```

优点：

- 适合节点性能不同的场景

缺点：

- 需要人工设置权重

## 7.5 综合评分策略

这是推荐重点实现的策略。

### 7.5.1 评分公式

```text
score = cpu_usage * 0.25
      + memory_usage * 0.15
      + bandwidth_usage_ratio * 0.40
      + connection_usage_ratio * 0.20
```

选择 score 最低的节点。

### 7.5.2 示例

```text
node-a:
  cpu = 60
  memory = 50
  bandwidth = 80
  connection = 70

score = 60*0.25 + 50*0.15 + 80*0.40 + 70*0.20
      = 15 + 7.5 + 32 + 14
      = 68.5
```

## 7.6 节点状态机

节点状态建议设计为：

```text
offline    节点离线
online     节点正常
degraded   节点负载较高，减少新连接
disabled   管理员手动禁用
```

状态转换：

```text
offline -> online    收到正常心跳
online -> degraded   资源超过阈值
degraded -> online   资源恢复正常
online -> offline    心跳超时
online -> disabled   管理员禁用
```

---

## 8. 监控设计

## 8.1 Prometheus 指标

代理节点建议暴露以下指标：

```text
proxy_node_connections_current
proxy_node_rx_bytes_total
proxy_node_tx_bytes_total
proxy_node_rx_mbps
proxy_node_tx_mbps
proxy_node_cpu_usage
proxy_node_memory_usage
proxy_user_rx_bytes_total
proxy_user_tx_bytes_total
proxy_user_rate_limited_total
proxy_connection_errors_total
```

控制中心建议暴露以下指标：

```text
control_nodes_online_total
control_nodes_offline_total
control_scheduler_events_total
control_user_traffic_used_bytes
control_api_requests_total
control_api_errors_total
```

## 8.2 Grafana 面板

建议设计以下面板：

| 面板 | 指标 |
|---|---|
| 节点总览 | 在线节点数、离线节点数、异常节点数 |
| 节点带宽 | 每个节点 RX/TX Mbps |
| 节点负载 | CPU、内存、连接数 |
| 用户流量 | 用户上传、下载、总使用量 |
| 限速事件 | 每个用户限速触发次数 |
| 调度事件 | 每种调度策略选择次数 |
| 故障事件 | 节点离线、恢复、降级次数 |

---

## 9. 部署架构

## 9.1 单机 Docker 实验环境

适合第一阶段学习。

```text
traffic-lab/
├── control-center
├── proxy-node-1
├── proxy-node-2
├── target-server
├── prometheus
└── grafana
```

拓扑：

```text
client -> proxy-node-1 -> target-server
client -> proxy-node-2 -> target-server
```

## 9.2 多机器实验环境

适合第二阶段验证真实网络。

```text
Machine 1:
  Control Center
  PostgreSQL
  Prometheus
  Grafana

Machine 2:
  Proxy Node A

Machine 3:
  Proxy Node B

Machine 4:
  Target Server
```

## 9.3 推荐端口规划

| 服务 | 端口 |
|---|---|
| Control Center API | 8000 |
| Control Center Web | 8080 |
| Proxy Node 1 | 10801 |
| Proxy Node 2 | 10802 |
| Prometheus | 9090 |
| Grafana | 3000 |
| Node Metrics | 9100 / 9200 |
| Target nginx | 8088 |
| iperf3 | 5201 |

---

## 10. 项目目录结构

```text
traffic-lab/
├── control-center/
│   ├── app/
│   │   ├── main.py
│   │   ├── api/
│   │   │   ├── nodes.py
│   │   │   ├── users.py
│   │   │   ├── traffic.py
│   │   │   └── scheduler.py
│   │   ├── models/
│   │   │   ├── user.py
│   │   │   ├── node.py
│   │   │   ├── traffic.py
│   │   │   └── policy.py
│   │   ├── services/
│   │   │   ├── scheduler_service.py
│   │   │   ├── traffic_service.py
│   │   │   └── node_service.py
│   │   └── config.py
│   ├── requirements.txt
│   └── Dockerfile
│
├── proxy-node/
│   ├── src/
│   │   ├── main.cpp
│   │   ├── tcp_proxy.cpp
│   │   ├── tcp_proxy.h
│   │   ├── connection_manager.cpp
│   │   ├── connection_manager.h
│   │   ├── token_bucket.cpp
│   │   ├── token_bucket.h
│   │   ├── traffic_counter.cpp
│   │   ├── traffic_counter.h
│   │   ├── policy_manager.cpp
│   │   ├── policy_manager.h
│   │   ├── heartbeat_reporter.cpp
│   │   └── heartbeat_reporter.h
│   ├── config/
│   │   └── node.yaml
│   ├── CMakeLists.txt
│   └── Dockerfile
│
├── deploy/
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   ├── grafana/
│   └── nginx.conf
│
├── database/
│   ├── schema.sql
│   └── init_data.sql
│
├── scripts/
│   ├── run_iperf_test.sh
│   ├── run_http_test.sh
│   ├── limit_bandwidth_tc.sh
│   ├── clear_tc.sh
│   └── collect_logs.sh
│
├── docs/
│   ├── architecture.md
│   ├── load-balance-strategy.md
│   ├── rate-limit-strategy.md
│   └── experiment-report-template.md
│
└── README.md
```

---

## 11. 实验路线设计

## 11.1 阶段一：基础 TCP 转发

目标：

```text
client -> proxy-node -> target-server
```

需要实现：

- TCP listener
- TCP connect target
- 双向数据转发
- 基础日志输出

测试工具：

```bash
iperf3 -s
iperf3 -c <proxy_node_ip> -p <proxy_port>
```

成功标准：

- 客户端可以通过代理访问目标服务
- 代理节点可以打印连接建立和断开日志
- 可以统计单条连接持续时间

## 11.2 阶段二：流量统计

需要实现：

- 每条连接 upload bytes
- 每条连接 download bytes
- 每个用户累计流量
- 每个节点累计流量
- 定时上报控制中心

成功标准：

- 控制中心可以看到用户流量
- 控制中心可以看到节点流量
- Grafana 可以显示流量曲线

## 11.3 阶段三：用户限速

需要实现：

- Token Bucket
- 用户级 max_rate
- burst 突发控制
- 限速事件统计

测试方法：

```bash
iperf3 -c <target> -t 30
```

观察：

- 不限速时吞吐量
- 限速 10 Mbps 时吞吐量
- burst 参数对短连接的影响

## 11.4 阶段四：节点负载调度

需要实现：

- Round Robin
- Least Connections
- Least Bandwidth
- 综合评分策略

实验方法：

- 同时启动多个 client
- 人为增加某个节点负载
- 观察新连接是否分配到低负载节点

## 11.5 阶段五：故障切换

实验方法：

```bash
docker stop proxy-node-1
```

观察：

- 控制中心多久发现节点离线
- 新连接是否分配到其他节点
- Grafana 是否显示告警
- scheduler_events 是否记录故障事件

---

## 12. 安全设计

## 12.1 用户认证

用户认证可以使用：

- token
- username + password
- JWT

实验阶段建议使用 token，简单直接。

### token 示例

```text
user001: 8b9f8f3e-xxxx-xxxx-xxxx-123456
```

客户端连接时携带 token，代理节点检查 token 是否有效。

## 12.2 节点认证

节点向控制中心注册和上报时必须携带 node_secret。

建议请求头：

```http
X-Node-Id: node-a
X-Node-Signature: <hmac-signature>
```

签名算法：

```text
signature = HMAC_SHA256(body, node_secret)
```

## 12.3 管理后台安全

即使是实验项目，也建议加入：

- 管理员强密码
- 操作日志
- 登录失败限制
- 配置文件权限控制
- 禁止把 token 打印到公开日志

## 12.4 日志策略

建议记录：

```text
user_id
node_id
connection_id
upload_bytes
download_bytes
start_time
end_time
error_code
```

不建议记录：

```text
完整 URL
用户通信内容
敏感请求参数
明文密码
完整 token
```

---

## 13. Linux 流量控制方案

## 13.1 应用层限速

第一版推荐应用层限速，便于理解算法。

优点：

- 逻辑清晰
- 易调试
- 可按用户细粒度控制

缺点：

- 性能不如内核层
- 高并发时要注意锁和定时器开销

## 13.2 tc 限速

Linux `tc` 可以做节点级和端口级限速。

示例：限制网卡出口 100Mbps：

```bash
sudo tc qdisc add dev eth0 root tbf rate 100mbit burst 32kbit latency 400ms
```

清理规则：

```bash
sudo tc qdisc del dev eth0 root
```

## 13.3 iptables/nftables 标记

可以配合 `iptables` 或 `nftables` 给不同流量打 mark，然后用 `tc` 做分类限速。

适合进阶实验：

```text
user001 -> mark 101 -> 10Mbps
user002 -> mark 102 -> 20Mbps
```

## 13.4 eBPF/XDP

eBPF/XDP 是高级方向，适合后续研究。

可研究内容：

- 内核态流量统计
- 高性能包过滤
- 连接级指标采集
- 低开销限速和观测

---

## 14. 性能优化方向

## 14.1 C++ 网络层优化

- 使用 epoll 边缘触发
- 减少内存拷贝
- 使用固定大小缓冲区
- 使用对象池管理连接对象
- 避免每个连接一个线程
- 采用事件循环 + 线程池
- 统计数据使用原子变量或分片计数器

## 14.2 数据上报优化

不要每条连接实时写数据库，可以采用批量上报。

推荐：

```text
节点本地每秒聚合
每 10 秒上报一次
控制中心批量写入数据库
Prometheus 走独立 metrics 接口
```

## 14.3 数据库优化

- traffic_logs 按时间分区
- 给 node_id、user_id、created_at 建索引
- 热数据放 Redis
- 历史数据定期归档
- 避免高频单行 update

---

## 15. 最小可行版本 MVP

## 15.1 MVP 功能边界

第一版只实现以下功能：

控制中心：

- 用户创建
- 节点注册
- 节点心跳
- 节点流量上报
- 简单调度策略
- Prometheus metrics

代理节点：

- TCP 转发
- 用户 token 验证
- 用户流量统计
- Token Bucket 限速
- 心跳上报
- 策略拉取

监控：

- 节点在线状态
- 节点连接数
- 节点 RX/TX
- 用户流量
- 限速事件

## 15.2 MVP 不包含

- 支付系统
- 商业套餐
- 工单系统
- 复杂订阅系统
- UDP 转发
- 高级伪装协议
- 多租户商业化权限系统

---

## 16. Docker Compose 实验设计

示例服务：

```yaml
version: "3.9"

services:
  control-center:
    build: ../control-center
    ports:
      - "8000:8000"
    depends_on:
      - postgres

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: trafficlab
      POSTGRES_USER: trafficlab
      POSTGRES_PASSWORD: trafficlab
    ports:
      - "5432:5432"

  proxy-node-1:
    build: ../proxy-node
    environment:
      NODE_ID: node-a
      CONTROL_CENTER_URL: http://control-center:8000
    ports:
      - "10801:1080"

  proxy-node-2:
    build: ../proxy-node
    environment:
      NODE_ID: node-b
      CONTROL_CENTER_URL: http://control-center:8000
    ports:
      - "10802:1080"

  target-server:
    image: nginx:latest
    ports:
      - "8088:80"

  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
```

---

## 17. 项目里程碑

| 阶段 | 时间 | 目标 |
|---|---|---|
| M1 | 第 1 周 | 实现单节点 TCP 转发 |
| M2 | 第 2 周 | 实现连接统计和用户流量统计 |
| M3 | 第 3 周 | 实现 Token Bucket 用户限速 |
| M4 | 第 4 周 | 实现控制中心和节点心跳 |
| M5 | 第 5 周 | 实现节点调度策略 |
| M6 | 第 6 周 | 接入 Prometheus + Grafana |
| M7 | 第 7 周 | 故障切换实验 |
| M8 | 第 8 周 | 整理实验报告和性能测试数据 |

---

## 18. 简历项目描述

### 中文版

项目名称：多节点代理流量调度实验平台

项目描述：

> 设计并实现一个多节点代理流量调度实验平台，包含控制中心、代理节点、用户限速、流量统计、节点心跳、负载调度和监控可视化等模块。代理节点基于 C++20 实现 TCP 转发和 Token Bucket 限速算法，控制中心负责节点管理、策略下发和负载调度，并通过 Prometheus + Grafana 展示节点带宽、连接数、CPU、内存和用户流量等指标。

技术栈：

```text
C++20 / Linux / TCP / epoll / FastAPI / PostgreSQL / Docker Compose / Prometheus / Grafana
```

项目亮点：

```text
- C++20 实现高性能 TCP 代理节点
- 支持用户级 Token Bucket 限速
- 支持节点心跳和健康检查
- 支持 Least Connections、Least Bandwidth 和综合评分调度
- 支持 Prometheus 指标采集和 Grafana 可视化
- 使用 Docker Compose 搭建完整实验环境
```

### 英文版

Project Name:

> Multi-node Traffic Scheduling and Proxy Control Platform

Description:

> Designed and implemented a multi-node proxy traffic scheduling platform for network flow control experiments. The system includes a control center, proxy nodes, traffic accounting, user-level rate limiting, node heartbeat, load-aware scheduling, and monitoring dashboards. Proxy nodes are implemented in C++20 with TCP forwarding and Token Bucket based rate limiting. The control center manages node status, policy distribution, and scheduling decisions, while Prometheus and Grafana are used for observability.

Tech Stack:

```text
C++20, Linux, TCP, epoll, FastAPI, PostgreSQL, Docker Compose, Prometheus, Grafana
```

Highlights:

```text
- Implemented a high-performance TCP proxy node in C++20
- Designed user-level Token Bucket rate limiting
- Built node heartbeat and health-check mechanism
- Implemented load-aware scheduling strategies
- Integrated Prometheus metrics and Grafana dashboards
- Deployed the whole platform with Docker Compose
```

---

## 19. 后续扩展方向

可以继续扩展：

- UDP 转发
- QUIC 实验
- eBPF 流量统计
- io_uring 网络转发
- Web 管理面板
- 用户优先级调度
- 动态权重调度
- 节点自动扩缩容
- 异常流量检测
- AI 流量预测
- 边缘网关部署
- 嵌入式 ARM 平台部署，例如 RK3568 / RK3588

---

## 20. 总结

这个项目最有价值的地方不是“搭建代理服务”本身，而是通过它系统学习：

```text
Linux 网络编程
高性能 TCP 转发
流量统计
限速算法
负载均衡
分布式节点管理
监控可观测性
Docker 化部署
系统安全设计
```

建议最终将项目包装为：

> 多节点流量调度实验平台

