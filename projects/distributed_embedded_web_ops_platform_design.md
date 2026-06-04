# 分布式嵌入式设备 Web 运维管理系统架构设计

> 设计目标：基于“最小输入信息 + Agent 自动注册 + Web 运维”的思路，使用 Go 语言设计一个可用于局域网和远程环境的嵌入式设备运行管理系统。系统面向 RK3568、RK3576、RK3588、Jetson、ARM Debian/Ubuntu、自研 BSP 设备等场景，支持设备发现、自动注册、状态监控、远程运维、日志采集、告警、固件版本管理和安全命令执行。

---

## 1. 设计背景

在嵌入式开发、BSP 调试、边缘设备管理、实验室设备运维中，常见问题包括：

- 开发板数量多，IP 经常变化。
- 每台设备的系统版本、内核版本、BSP 版本、驱动状态不一致。
- 需要频繁查看 CPU、内存、磁盘、温度、GPU、USB、WiFi、音频、摄像头等状态。
- 需要远程执行少量安全命令，例如查看日志、重启应用、设置音量、检测 USB 设备。
- 需要降低人工录入信息量，不希望每台设备都手动填写大量配置。
- 局域网内设备可以直接访问，但远程设备可能位于 NAT 或防火墙之后。

因此，本系统采用：

```text
最小输入信息 + Agent 自动注册 + Web 运维管理
```

核心思想是：

```text
用户只输入 IP / 首次授权码 / 设备名称等极少信息
        ↓
设备端 Agent 自动采集完整设备信息
        ↓
Agent 主动向 Web 管理平台注册
        ↓
后端建立设备身份、状态、指标、日志和命令通道
        ↓
Web 前端统一管理和运维
```

---

## 2. 系统设计目标

### 2.1 功能目标

系统需要支持：

1. 设备自动注册。
2. 设备在线 / 离线检测。
3. 局域网设备发现。
4. 远程设备主动连接。
5. 设备基础信息采集。
6. CPU / 内存 / 磁盘 / 温度 / 网络指标采集。
7. USB / WiFi / 声卡 / GPU / 摄像头等嵌入式专项检测。
8. 日志采集与查询。
9. 安全远程命令执行。
10. 应用进程管理。
11. 固件 / BSP / 镜像版本记录。
12. 告警与通知。
13. 多用户权限控制。
14. 高并发接入。
15. 高可用部署。

### 2.2 非功能目标

| 目标 | 说明 |
|---|---|
| 最小输入 | 用户尽量只输入 IP、设备名称或一次性注册码 |
| 自动识别 | Agent 自动采集设备硬件、系统、网络、驱动信息 |
| 高并发 | 支持大量 Agent 同时在线和上报 |
| 高稳定 | 后端服务可水平扩展，Agent 自动重连 |
| 安全 | TLS 加密、Token 认证、命令白名单、权限控制 |
| 可观测 | 日志、指标、链路追踪、告警完整 |
| 易部署 | 后端支持 Docker Compose / Kubernetes，Agent 支持 systemd / Yocto 集成 |
| 可扩展 | 后续可增加 OTA、文件分发、远程调试、自动化测试 |

---

## 3. 最小输入信息设计

### 3.1 不同场景下的最小输入

| 场景 | 用户最小输入 | 系统自动补全的信息 |
|---|---|---|
| 局域网管理 | 设备 IP | hostname、MAC、系统版本、内核版本、Agent 状态 |
| 远程管理 | 设备注册码 / 设备连接地址 | 公网连接状态、设备身份、Agent 通道 |
| 首次接入 | 一次性注册码 | device_id、Token、证书、设备信息 |
| 已安装 Agent | 无需手动输入 | Agent 自动注册、自动心跳、自动上报 |
| 临时接入 | IP + SSH 账号或密钥 | 可尝试远程安装 Agent |

### 3.2 推荐最小输入模型

第一版建议支持两种模式：

#### 模式 A：局域网 IP 接入

用户只输入：

```text
设备 IP
```

后端尝试：

```text
1. Ping / TCP 探测设备是否在线
2. 检测 Agent 端口是否开放
3. 如果 Agent 已存在，拉取设备信息
4. 如果 Agent 不存在，提示用户安装 Agent
```

#### 模式 B：Agent 自动注册

用户只需要在设备端配置：

```yaml
server_url: "https://manager.example.com"
register_code: "一次性注册码"
```

Agent 首次启动后自动完成：

```text
1. 向后端提交 register_code
2. 后端校验注册码
3. 后端分配 device_id 和 agent_token
4. Agent 保存身份信息
5. 后续自动上报心跳和指标
```

### 3.3 最小但必要的信息

虽然用户输入可以很少，但系统内部至少要维护以下信息：

| 信息 | 是否用户手动输入 | 来源 | 用途 |
|---|---|---|---|
| device_id | 否 | Agent 自动生成 / 后端分配 | 设备唯一标识 |
| IP 地址 | 可选 | Agent 上报 / 扫描发现 | 网络访问 |
| MAC 地址 | 否 | Agent 采集 | 辅助识别设备 |
| hostname | 否 | Agent 采集 | 页面展示 |
| board_type | 否 | Agent 采集 / 用户标注 | 判断设备类型 |
| arch | 否 | Agent 采集 | 判断二进制兼容性 |
| OS 版本 | 否 | Agent 采集 | 判断运维命令兼容性 |
| Kernel 版本 | 否 | Agent 采集 | BSP / 驱动分析 |
| Agent Token | 否 | 注册时生成 | 身份认证 |
| Agent 版本 | 否 | Agent 上报 | 升级管理 |

---

## 4. 系统整体架构

### 4.1 总体架构图

![分布式嵌入式设备 Web 运维管理系统架构图](a_clean_technical_infographic_architecture_diagr.png)

### 4.2 逻辑架构

```text
┌──────────────────────────────────────────────┐
│                  Web 前端                     │
│ Vue / React / TypeScript / ECharts            │
│ 设备管理 / 监控 / 日志 / 命令 / 固件 / 告警     │
└─────────────────────┬────────────────────────┘
                      │ HTTPS / WebSocket
                      ▼
┌──────────────────────────────────────────────┐
│              API Gateway / Load Balancer      │
│ Nginx / Traefik / Kong / Envoy                │
│ TLS 终止 / 路由 / 限流 / 鉴权前置              │
└─────────────────────┬────────────────────────┘
                      │
                      ▼
┌──────────────────────────────────────────────┐
│               Go 后端服务集群                  │
│ Gin / Echo / Fiber + gRPC + WebSocket          │
│                                              │
│ - 设备管理服务                                │
│ - Agent 注册服务                              │
│ - 心跳服务                                    │
│ - 指标接收服务                                │
│ - 命令下发服务                                │
│ - 日志服务                                    │
│ - 告警服务                                    │
│ - 固件管理服务                                │
│ - 用户权限服务                                │
└──────────────┬───────────────┬───────────────┘
               │               │
               ▼               ▼
┌──────────────────────┐  ┌──────────────────────┐
│ PostgreSQL / MySQL    │  │ Redis Cluster         │
│ 设备资产 / 用户 / 命令 │  │ 在线状态 / 缓存 / 限流 │
└──────────────────────┘  └──────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│                时序与日志系统                  │
│ Prometheus / VictoriaMetrics / Elasticsearch  │
│ Loki / OpenSearch                             │
└──────────────────────────────────────────────┘
               ▲
               │ HTTPS / WSS / gRPC / MQTT 可选
               │
┌──────────────┴──────────────┬───────────────┬───────────────┐
│         RK3568 Agent         │  RK3588 Agent │  Jetson Agent  │
│ Debian / Ubuntu / Yocto      │  BSP 系统      │  Ubuntu / L4T  │
└──────────────────────────────┴───────────────┴───────────────┘
```

---

## 5. 核心组件设计

## 5.1 Web 前端

### 技术选型

| 模块 | 推荐 |
|---|---|
| 框架 | Vue 3 + TypeScript 或 React + TypeScript |
| UI | Naive UI / Element Plus / Ant Design |
| 图表 | ECharts |
| 状态管理 | Pinia / Zustand / Redux Toolkit |
| 实时通信 | WebSocket / SSE |
| 构建工具 | Vite |

### 页面设计

```text
frontend/
├── 设备总览页
├── 设备详情页
├── 指标监控页
├── 日志查看页
├── 命令执行页
├── 固件管理页
├── 告警中心
├── 用户与权限管理
└── 系统配置页
```

### 设备总览页字段

| 字段 | 示例 |
|---|---|
| 设备名 | UltrasoundOS-01 |
| IP | 192.168.40.23 |
| 设备类型 | RK3588 |
| 系统 | Debian 11 |
| 内核 | 5.10.160 |
| Agent 版本 | v1.0.0 |
| 在线状态 | Online |
| CPU | 23% |
| 内存 | 61% |
| 温度 | 58°C |
| 最近心跳 | 5 秒前 |

---

## 5.2 Go API Gateway / 后端服务

### 技术选型

| 功能 | 推荐技术 |
|---|---|
| Web 框架 | Gin / Echo / Fiber |
| RPC | gRPC |
| WebSocket | nhooyr.io/websocket / Gorilla WebSocket |
| 数据库访问 | sqlx / GORM / pgx |
| 配置 | Viper |
| 日志 | Zap / Zerolog |
| 认证 | JWT + Refresh Token |
| 消息队列 | NATS / Kafka / RabbitMQ |
| 指标 | Prometheus Client |
| 链路追踪 | OpenTelemetry |

### 后端服务模块

```text
backend/
├── cmd/server/main.go
├── internal/api
│   ├── auth_api.go
│   ├── device_api.go
│   ├── agent_api.go
│   ├── metric_api.go
│   ├── command_api.go
│   ├── log_api.go
│   ├── firmware_api.go
│   └── alert_api.go
├── internal/service
│   ├── auth_service.go
│   ├── device_service.go
│   ├── register_service.go
│   ├── heartbeat_service.go
│   ├── metric_service.go
│   ├── command_service.go
│   ├── log_service.go
│   ├── firmware_service.go
│   └── alert_service.go
├── internal/model
│   ├── user.go
│   ├── device.go
│   ├── agent.go
│   ├── metric.go
│   ├── command.go
│   ├── log.go
│   └── alert.go
├── internal/repository
│   ├── postgres_device_repo.go
│   ├── redis_online_repo.go
│   └── metric_repo.go
├── internal/ws
│   ├── hub.go
│   └── client.go
└── pkg
    ├── crypto
    ├── token
    ├── config
    └── logger
```

---

## 5.3 设备 Agent

### Agent 设计目标

Agent 是系统的关键。它运行在嵌入式设备上，负责：

```text
1. 自动注册
2. 身份认证
3. 心跳上报
4. 指标采集
5. 日志采集
6. 命令执行
7. 文件上传 / 下载
8. 固件版本上报
9. 断线重连
10. 自身升级
```

### Agent 技术选型

建议使用 Go 编写，原因：

```text
1. 静态编译，部署简单
2. 交叉编译 ARM 方便
3. 并发能力强
4. 适合网络长连接
5. 适合做 systemd 常驻服务
6. 资源占用可控
```

### Agent 目录结构

```text
agent/
├── cmd/agent/main.go
├── internal/config/config.go
├── internal/register/register.go
├── internal/heartbeat/heartbeat.go
├── internal/collector
│   ├── basic.go
│   ├── cpu.go
│   ├── memory.go
│   ├── disk.go
│   ├── thermal.go
│   ├── network.go
│   ├── usb.go
│   ├── wifi.go
│   ├── audio.go
│   ├── gpu.go
│   └── process.go
├── internal/command
│   ├── dispatcher.go
│   ├── whitelist.go
│   └── executor.go
├── internal/logs
│   ├── dmesg.go
│   ├── journal.go
│   └── app_log.go
├── internal/client
│   ├── http_client.go
│   ├── ws_client.go
│   └── retry.go
└── pkg
    ├── machineid
    ├── shellsafe
    └── crypto
```

### Agent 配置文件

路径建议：

```text
/etc/embedded-agent/config.yaml
```

示例：

```yaml
server:
  url: "https://manager.example.com"
  websocket_url: "wss://manager.example.com/agent/ws"

agent:
  device_name: "rk3588-lab-01"
  register_code: "FIRST-REGISTER-CODE"
  data_dir: "/var/lib/embedded-agent"
  log_file: "/var/log/embedded-agent.log"

security:
  tls_verify: true
  token_file: "/var/lib/embedded-agent/token"
  device_cert: "/var/lib/embedded-agent/device.crt"
  device_key: "/var/lib/embedded-agent/device.key"

collector:
  heartbeat_interval_sec: 5
  metrics_interval_sec: 10
  logs_interval_sec: 60

command:
  enable_remote_command: true
  allow_reboot: false
  allow_shell: false
```

---

## 6. 自动注册流程设计

### 6.1 首次注册流程

```text
Agent 启动
  ↓
读取 /etc/embedded-agent/config.yaml
  ↓
读取本地 device_id / token
  ↓
如果不存在，进入首次注册流程
  ↓
采集基础设备信息
  ↓
携带 register_code 请求后端注册
  ↓
后端验证 register_code
  ↓
后端生成 device_id / agent_token / device_secret
  ↓
Agent 保存到 /var/lib/embedded-agent/
  ↓
Agent 开始心跳与指标上报
```

### 6.2 注册请求

```http
POST /api/v1/agents/register
```

请求体：

```json
{
  "register_code": "FIRST-REGISTER-CODE",
  "hostname": "UltrasoundOS",
  "machine_id": "7f4c9a8e9d0f4c1d",
  "mac_addresses": ["a0:ad:9f:d3:35:3a"],
  "arch": "aarch64",
  "os_name": "Debian GNU/Linux 11",
  "kernel_version": "5.10.160",
  "board_type": "RK3588",
  "agent_version": "1.0.0"
}
```

响应：

```json
{
  "device_id": "dev-rk3588-000001",
  "agent_token": "eyJhbGciOi...",
  "server_time": 1760000000,
  "config_version": 1
}
```

### 6.3 后续认证

后续所有请求携带：

```http
Authorization: Bearer <agent_token>
X-Device-ID: dev-rk3588-000001
```

---

## 7. 心跳与在线状态设计

### 7.1 心跳上报

```http
POST /api/v1/agents/heartbeat
```

```json
{
  "device_id": "dev-rk3588-000001",
  "timestamp": 1760000000,
  "uptime_sec": 3600,
  "agent_version": "1.0.0",
  "app_status": "running"
}
```

### 7.2 在线状态判断

后端 Redis 保存：

```text
online:device:{device_id} = last_seen_timestamp
TTL = heartbeat_interval * 3
```

例如心跳间隔 5 秒，则 TTL 设为 15 秒或 20 秒。

判断逻辑：

```text
如果 Redis key 存在 → Online
如果 Redis key 不存在 → Offline
```

### 7.3 稳定性策略

| 问题 | 处理策略 |
|---|---|
| 网络抖动 | Agent 指数退避重试 |
| 后端重启 | Agent 自动重连 |
| 心跳偶发失败 | 允许 3 次失败后判定离线 |
| 大量 Agent 同时重连 | 随机 jitter 延迟 |
| Redis 故障 | 降级到数据库 last_seen |

---

## 8. 指标采集设计

### 8.1 基础指标

Agent 周期采集：

| 指标 | Linux 来源 |
|---|---|
| CPU 使用率 | `/proc/stat` |
| 内存使用率 | `/proc/meminfo` |
| 磁盘使用率 | `statfs` / `df` |
| 网络流量 | `/proc/net/dev` |
| 系统负载 | `/proc/loadavg` |
| 运行时间 | `/proc/uptime` |

### 8.2 嵌入式专项指标

| 类型 | 采集命令 / 文件 | 用途 |
|---|---|---|
| 温度 | `/sys/class/thermal/thermal_zone*/temp` | CPU/GPU/NPU 温度 |
| GPU 频率 | `/sys/class/devfreq/*/cur_freq` | 判断 GPU 频率 |
| GPU 负载 | `/sys/class/devfreq/*/load` | 判断 GPU 是否工作 |
| DRI 设备 | `ls /dev/dri` | 判断图形设备是否存在 |
| Mali 设备 | `ls /dev/mali*` | 判断 Mali 驱动是否存在 |
| USB | `lsusb` | 检测外设 |
| WiFi | `iw dev`, `ip link`, `lsmod` | 检测无线模块 |
| 声卡 | `aplay -l`, `amixer -c 0` | 检测音频状态 |
| 摄像头 | `ls /dev/video*`, `v4l2-ctl --list-devices` | 检测摄像头 |

### 8.3 指标上报接口

```http
POST /api/v1/agents/metrics
```

```json
{
  "device_id": "dev-rk3588-000001",
  "timestamp": 1760000000,
  "cpu_usage": 23.5,
  "memory_usage": 61.2,
  "disk_usage": 72.1,
  "load_avg_1m": 0.43,
  "temperature": 58.3,
  "gpu_load": 34.0,
  "network_rx_bytes": 1024000,
  "network_tx_bytes": 2048000
}
```

---

## 9. 远程命令设计

### 9.1 设计原则

远程命令是高风险功能，不能直接开放任意 shell。

不推荐：

```json
{
  "cmd": "rm -rf /"
}
```

推荐：

```json
{
  "action": "set_audio_volume",
  "params": {
    "card": 0,
    "control": "HP",
    "value": 34
  }
}
```

后端只允许预定义动作，Agent 只执行白名单命令。

### 9.2 命令白名单

| Action | 实际操作 | 风险级别 |
|---|---|---|
| get_basic_info | 读取系统基础信息 | 低 |
| get_dmesg | 获取 dmesg 日志 | 低 |
| get_usb_devices | 执行 lsusb | 低 |
| get_audio_status | 执行 amixer / aplay | 低 |
| set_audio_volume | 执行 amixer 设置音量 | 中 |
| get_gpu_status | 检测 GPU / DRI / Mali | 低 |
| restart_app | 重启指定应用服务 | 中 |
| restart_agent | 重启 Agent | 中 |
| reboot_device | 重启设备 | 高，默认关闭 |
| upload_log_bundle | 打包上传日志 | 中 |

### 9.3 命令下发方式

第一版可以用 HTTP 轮询：

```text
Agent 每 3 秒请求 /api/v1/agents/tasks/pull
```

第二版使用 WebSocket：

```text
后端通过 WebSocket 直接向 Agent 下发任务
```

第三版使用 gRPC 双向流：

```text
高并发、强结构化、大规模设备管理
```

### 9.4 命令执行流程

```text
Web 用户点击“设置 HP 音量为 34”
        ↓
后端检查用户权限
        ↓
后端创建 command_task
        ↓
后端通过 WebSocket / 轮询交给 Agent
        ↓
Agent 校验 action 是否在白名单
        ↓
Agent 执行安全封装后的命令
        ↓
Agent 返回 stdout / stderr / exit_code
        ↓
后端保存执行结果
        ↓
Web 前端展示结果
```

### 9.5 示例：设置 HP 音量

Web 请求：

```http
POST /api/v1/devices/dev-rk3588-000001/commands
```

```json
{
  "action": "set_audio_volume",
  "params": {
    "card": 0,
    "control": "HP",
    "value": 34
  }
}
```

Agent 内部执行逻辑：

```text
1. 检查 action == set_audio_volume
2. 检查 card 只能是 0~8 的整数
3. 检查 control 只能是 HP / Speaker / Headphone 等允许项
4. 检查 value 在合法范围内
5. 执行 amixer -c 0 sset 'HP' 34
6. 执行 amixer -c 0 get 'HP' 验证结果
```

---

## 10. 日志采集设计

### 10.1 日志类型

| 日志类型 | 来源 |
|---|---|
| kernel | `dmesg` |
| systemd | `journalctl` |
| app | 应用日志文件 |
| agent | `/var/log/embedded-agent.log` |
| audio | amixer / PulseAudio 相关日志 |
| gpu | EGL / OpenGL / Mali 相关日志 |
| wifi | WiFi 驱动和连接日志 |

### 10.2 日志采集策略

第一版：按需拉取。

```text
用户点击“查看最近 200 行 dmesg”
        ↓
后端下发 get_dmesg action
        ↓
Agent 返回日志
```

第二版：周期采集关键日志。

```text
Agent 每 60 秒上报关键错误日志
```

第三版：日志流式传输。

```text
Agent tail -f 应用日志，通过 WebSocket 推送给后端
```

### 10.3 常见嵌入式问题关键词

系统可以自动标记这些关键词：

```text
libEGL
llvmpipe
mali
failed to load driver
DRI2
DRI3
segmentation fault
amixer
alsa
pulseaudio
usb
wifi
aic8800
cfg80211
kernel panic
Out of memory
```

---

## 11. 数据库设计

### 11.1 devices 表

```sql
CREATE TABLE devices (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) UNIQUE NOT NULL,
    device_name VARCHAR(128),
    hostname VARCHAR(128),
    ip_address VARCHAR(64),
    mac_address VARCHAR(128),
    board_type VARCHAR(64),
    arch VARCHAR(64),
    os_name VARCHAR(128),
    os_version VARCHAR(128),
    kernel_version VARCHAR(128),
    bsp_version VARCHAR(128),
    agent_version VARCHAR(64),
    status VARCHAR(32) DEFAULT 'offline',
    last_seen TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 11.2 agent_credentials 表

```sql
CREATE TABLE agent_credentials (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) UNIQUE NOT NULL,
    token_hash TEXT NOT NULL,
    cert_fingerprint VARCHAR(256),
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 11.3 device_metrics 表

```sql
CREATE TABLE device_metrics (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    cpu_usage DOUBLE PRECISION,
    memory_usage DOUBLE PRECISION,
    disk_usage DOUBLE PRECISION,
    load_avg_1m DOUBLE PRECISION,
    temperature DOUBLE PRECISION,
    gpu_load DOUBLE PRECISION,
    network_rx_bytes BIGINT,
    network_tx_bytes BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 11.4 command_tasks 表

```sql
CREATE TABLE command_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(128) UNIQUE NOT NULL,
    device_id VARCHAR(128) NOT NULL,
    action VARCHAR(128) NOT NULL,
    params JSONB,
    status VARCHAR(32) DEFAULT 'pending',
    stdout TEXT,
    stderr TEXT,
    exit_code INT,
    created_by VARCHAR(128),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP
);
```

### 11.5 device_logs 表

```sql
CREATE TABLE device_logs (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128) NOT NULL,
    log_type VARCHAR(64),
    level VARCHAR(32),
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 11.6 alerts 表

```sql
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(128),
    alert_type VARCHAR(128),
    severity VARCHAR(32),
    message TEXT,
    status VARCHAR(32) DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);
```

---

## 12. 高并发设计

### 12.1 并发压力来源

系统的高并发主要来自：

```text
1. 大量 Agent 同时心跳
2. 大量 Agent 同时上报指标
3. WebSocket 长连接
4. 日志上传
5. 命令任务结果回传
6. 前端实时监控页面订阅
```

### 12.2 高并发架构策略

| 层级 | 策略 |
|---|---|
| 接入层 | Nginx / Traefik 负载均衡，多后端实例 |
| API 层 | Go 无状态服务，水平扩展 |
| 心跳 | Redis TTL 存在线状态，避免频繁写数据库 |
| 指标 | 批量写入，异步队列削峰 |
| 日志 | 消息队列 + Elasticsearch / Loki |
| WebSocket | 独立 ws-gateway 服务，可水平扩展 |
| 数据库 | 主从、读写分离、索引优化、冷热数据分离 |
| 缓存 | Redis Cluster 缓存设备状态和用户会话 |

### 12.3 心跳优化

不要每次心跳都写 PostgreSQL。

推荐：

```text
Agent 心跳 → Go 后端 → Redis 设置 TTL
```

每隔一定时间再异步更新数据库：

```text
Redis 在线状态 → 后台任务 → 批量更新 devices.last_seen
```

### 12.4 指标写入优化

小规模：

```text
Agent → Go API → PostgreSQL
```

中大规模：

```text
Agent → Go API → NATS/Kafka → Metrics Worker → TimescaleDB / VictoriaMetrics
```

### 12.5 日志写入优化

```text
Agent → Go API → Kafka/NATS → Log Worker → Elasticsearch / Loki
```

---

## 13. 高可用设计

### 13.1 后端无状态化

Go API Server 不保存本地状态，状态放到：

```text
PostgreSQL / Redis / 消息队列 / 对象存储
```

这样可以部署多个实例：

```text
api-server-1
api-server-2
api-server-3
```

由 Nginx / Traefik 负载均衡。

### 13.2 Agent 自动重连

Agent 需要具备：

```text
1. 启动自动连接
2. 连接失败重试
3. 指数退避
4. 随机 jitter
5. 本地缓存未上报数据
6. 网络恢复后补发
```

### 13.3 服务降级

| 故障 | 降级策略 |
|---|---|
| Redis 故障 | 退化为数据库 last_seen 判断在线 |
| PostgreSQL 故障 | Agent 本地缓存，后端返回临时不可用 |
| 日志系统故障 | 暂停日志写入，不影响心跳 |
| 指标系统故障 | 丢弃非关键指标，保留心跳 |
| WebSocket 故障 | 退化为 HTTP 轮询 |

---

## 14. 安全设计

### 14.1 通信安全

| 通信链路 | 安全措施 |
|---|---|
| Web 前端 → 后端 | HTTPS + JWT |
| Agent → 后端 | HTTPS / WSS + Agent Token |
| 后端内部服务 | mTLS 可选 |
| 数据库存储 | 最小权限账号 + 敏感字段加密 |

### 14.2 Agent 身份认证

首次注册使用：

```text
register_code
```

注册成功后使用：

```text
agent_token
```

更高安全版本可使用：

```text
设备证书 + mTLS
```

### 14.3 命令安全

必须遵守：

```text
1. 不开放任意 shell
2. 使用 action 白名单
3. 参数类型严格校验
4. 高危操作需要二次确认
5. 所有命令记录审计日志
6. 不允许前端直接传 shell 字符串
```

### 14.4 权限模型

推荐 RBAC：

| 角色 | 权限 |
|---|---|
| Admin | 全部权限 |
| Operator | 查看设备、执行低中风险命令 |
| Viewer | 只能查看设备和指标 |
| Auditor | 查看审计日志 |

---

## 15. 局域网与远程管理设计

### 15.1 局域网管理

局域网内可以支持：

```text
1. 输入 IP 添加设备
2. 扫描网段发现设备
3. 检测 Agent 端口
4. 通过 Agent 管理设备
5. 如果没有 Agent，提示安装 Agent
```

### 15.2 远程管理

远程设备通常在 NAT 后面，Web 后端无法主动连接设备。

推荐方案：

```text
Agent 主动连接 Server
```

也就是：

```text
远程设备 Agent → 公网 Web 管理平台
```

后端命令通过已有连接下发。

### 15.3 不建议只依赖 SSH

SSH 适合临时运维，但不适合作为大规模设备管理主通道。

原因：

```text
1. 需要账号密码或密钥
2. NAT 后面无法主动连接
3. 安全风险高
4. 不适合持续心跳和指标采集
5. 不适合前端实时状态展示
```

推荐：

```text
Agent 为主，SSH 为辅
```

---

## 16. 设备检测能力设计

### 16.1 基础检测

```bash
hostname
uname -a
cat /etc/os-release
cat /proc/cpuinfo
cat /proc/meminfo
df -h
ip addr
```

### 16.2 GPU 检测

```bash
ls /dev/dri
ls /dev/mali* 2>/dev/null
cat /sys/class/devfreq/*/load 2>/dev/null
cat /sys/class/devfreq/*/cur_freq 2>/dev/null
glxinfo | grep "OpenGL renderer" 2>/dev/null
glmark2-es2 --off-screen 2>/dev/null
```

用于判断：

```text
1. 是否有 DRI 设备
2. 是否有 Mali 设备
3. 是否是 llvmpipe 软件渲染
4. GPU 是否有负载
5. OpenGL / EGL 是否正常
```

### 16.3 USB 检测

```bash
lsusb
dmesg | grep -i usb
```

可用于识别：

```text
TP-Link USB WiFi
AIC8800 WiFi 模块
触摸屏
摄像头
GD32 设备
Jetson USB 设备
```

### 16.4 WiFi 检测

```bash
iw dev
ip link
nmcli dev status 2>/dev/null
lsmod | grep -iE "aic|wifi|wlan|cfg80211"
```

### 16.5 音频检测

```bash
aplay -l
cat /proc/asound/cards
amixer -c 0
amixer -c 0 get 'HP' 2>/dev/null
amixer -c 0 get 'Speaker' 2>/dev/null
pactl list sinks short 2>/dev/null
```

可支持 Web 按钮：

```text
设置 HP 音量为 34
设置 Speaker 音量为 39
查看声卡状态
查看 PulseAudio Sink
```

---

## 17. API 设计

### 17.1 用户登录

```http
POST /api/v1/auth/login
```

### 17.2 Agent 注册

```http
POST /api/v1/agents/register
```

### 17.3 Agent 心跳

```http
POST /api/v1/agents/heartbeat
```

### 17.4 Agent 指标上报

```http
POST /api/v1/agents/metrics
```

### 17.5 获取设备列表

```http
GET /api/v1/devices
```

### 17.6 获取设备详情

```http
GET /api/v1/devices/{device_id}
```

### 17.7 创建远程命令

```http
POST /api/v1/devices/{device_id}/commands
```

### 17.8 Agent 拉取任务

```http
GET /api/v1/agents/tasks/pull
```

### 17.9 Agent 上报任务结果

```http
POST /api/v1/agents/tasks/{task_id}/result
```

### 17.10 获取日志

```http
GET /api/v1/devices/{device_id}/logs
```

---

## 18. 部署架构

### 18.1 开发环境

```text
Docker Compose
├── go-api-server
├── postgres
├── redis
├── frontend
└── prometheus 可选
```

### 18.2 生产环境

```text
Nginx / Traefik
    ↓
Go API Server x N
    ↓
PostgreSQL 主从 / 云数据库
Redis Cluster
NATS / Kafka
Prometheus / VictoriaMetrics
Loki / Elasticsearch
MinIO / S3
```

### 18.3 Agent 部署

Linux 设备上：

```text
/usr/local/bin/embedded-agent
/etc/embedded-agent/config.yaml
/var/lib/embedded-agent/
/var/log/embedded-agent.log
/etc/systemd/system/embedded-agent.service
```

systemd service：

```ini
[Unit]
Description=Embedded Device Management Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/embedded-agent -config /etc/embedded-agent/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 19. Yocto / BSP 集成设计

如果要将 Agent 固化进系统镜像，可创建 Yocto recipe：

```text
meta-yourlayer/
└── recipes-support/
    └── embedded-agent/
        ├── embedded-agent_1.0.bb
        └── files/
            ├── embedded-agent
            ├── embedded-agent.service
            └── config.yaml
```

recipe 示例：

```bitbake
SUMMARY = "Embedded device management agent"
LICENSE = "CLOSED"
SRC_URI = "file://embedded-agent \
           file://embedded-agent.service \
           file://config.yaml"

inherit systemd

SYSTEMD_SERVICE:${PN} = "embedded-agent.service"
SYSTEMD_AUTO_ENABLE:${PN} = "enable"

do_install() {
    install -d ${D}${bindir}
    install -m 0755 ${WORKDIR}/embedded-agent ${D}${bindir}/embedded-agent

    install -d ${D}${sysconfdir}/embedded-agent
    install -m 0644 ${WORKDIR}/config.yaml ${D}${sysconfdir}/embedded-agent/config.yaml

    install -d ${D}${systemd_system_unitdir}
    install -m 0644 ${WORKDIR}/embedded-agent.service ${D}${systemd_system_unitdir}/embedded-agent.service
}
```

image recipe：

```bitbake
IMAGE_INSTALL:append = " embedded-agent"
```

---

## 20. 开发路线图

### 阶段 1：MVP

目标：先跑通设备自动注册和在线状态。

功能：

```text
1. Go 后端
2. Vue 前端
3. PostgreSQL
4. Redis
5. Agent 自动注册
6. Agent 心跳
7. 设备列表页
8. 设备详情页
```

### 阶段 2：指标监控

```text
CPU
内存
磁盘
温度
网络
系统负载
```

### 阶段 3：嵌入式专项检测

```text
GPU / OpenGL / Mali / llvmpipe
USB 设备
WiFi 模块
声卡状态
摄像头
触摸屏
```

### 阶段 4：远程命令

```text
命令白名单
任务下发
执行结果回传
审计日志
```

### 阶段 5：日志与告警

```text
dmesg
journalctl
应用日志
关键词告警
设备离线告警
指标阈值告警
```

### 阶段 6：固件与 OTA

```text
BSP 版本记录
系统镜像版本记录
应用版本记录
Agent 升级
固件升级
回滚机制
```

---

## 21. 最终推荐技术栈

| 层级 | 推荐技术 |
|---|---|
| 前端 | Vue 3 + TypeScript + Naive UI + ECharts |
| 后端 | Go + Gin / Echo |
| 实时通信 | WebSocket，后续可升级 gRPC 双向流 |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 时序指标 | Prometheus / VictoriaMetrics |
| 日志 | Loki / Elasticsearch |
| 消息队列 | NATS / Kafka |
| 对象存储 | MinIO / S3 |
| 部署 | Docker Compose / Kubernetes |
| Agent | Go 静态编译 + systemd |
| 安全 | HTTPS + JWT + Agent Token + 命令白名单 |

---

## 22. 总结

本系统的核心不是让用户填写大量设备信息，而是让设备端 Agent 自动完成：

```text
自动注册
自动识别
自动心跳
自动采集
自动上报
自动重连
```

用户侧只需要最小输入：

```text
局域网场景：输入 IP
远程场景：配置一次性注册码
已安装 Agent 场景：无需输入
```

最终形成一个适合嵌入式设备、开发板、BSP 测试设备和边缘设备的分布式 Web 运维管理平台。

推荐第一版先实现：

```text
Go 后端 + Vue 前端 + PostgreSQL + Redis + Go Agent
```

先完成：

```text
设备注册
心跳
设备列表
基础指标
安全命令白名单
```

再逐步增加：

```text
GPU 检测
USB / WiFi / 声卡检测
日志采集
告警
固件版本管理
OTA 升级
```
