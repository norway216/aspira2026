# Go 语言局域网设备检测与运维管理 Web 系统架构设计

> 目标：设计一个使用 Go 语言开发的 Web 系统，用于周期性扫描当前局域网内在线设备，获取设备 IP、MAC、在线状态、在线时长、流量指标，并通过 Web 页面进行可视化管理。同时系统具备用户管理、设备管理、告警、权限控制、日志审计等外围功能。

---

## 1. 系统定位

这个系统可以理解为一个轻量级的 **局域网资产发现 + 在线状态监控 + 流量可视化 + Web 运维管理平台**。

它不是传统意义上的入侵工具，也不是网络攻击系统，而是面向公司内部、实验室、嵌入式设备、开发板、测试设备的资产管理系统。

适合场景：

- 公司内部局域网设备管理
- 嵌入式开发板在线状态检测
- 医疗设备、超声设备、工控设备状态观察
- 测试实验室设备资产管理
- 局域网内未知设备发现
- 网络流量异常设备排序
- 设备在线时长统计
- 轻量级 NMS 网络管理系统

---

## 2. 核心功能需求

### 2.1 在线设备扫描

系统启动后，自动周期性扫描当前局域网中的在线设备。

扫描方式包括：

| 扫描方式 | 作用 | 优点 | 缺点 |
|---|---|---|---|
| ARP 扫描 | 发现同一二层局域网内设备 | 快速，能获取 MAC | 只能扫描当前网段或二层可达网络 |
| Ping 扫描 | 检测 IP 是否在线 | 实现简单 | 有些设备禁 ICMP |
| TCP 端口探测 | 判断设备服务是否在线 | 对禁 ping 设备有效 | 扫描成本较高 |
| DHCP 租约读取 | 获取已分配设备 | 准确度较高 | 依赖路由器或 DHCP 服务权限 |
| Agent 主动上报 | 设备主动注册 | 准确、实时、可扩展 | 需要在设备端部署 Agent |

推荐初期采用：

```text
ARP 扫描 + Ping 扫描 + TCP 探测
```

后期升级为：

```text
主动扫描 + Agent 上报 + SNMP/NetFlow/sFlow 数据采集
```

---

### 2.2 设备信息采集

每个在线设备至少采集以下信息：

| 字段 | 说明 |
|---|---|
| IP 地址 | 当前设备的 IPv4 地址 |
| MAC 地址 | 网卡物理地址 |
| 主机名 | 可通过 DNS、mDNS、NetBIOS、Agent 获取 |
| 厂商信息 | 根据 MAC OUI 判断设备厂商 |
| 首次发现时间 | 第一次扫描到该设备的时间 |
| 最近在线时间 | 最近一次扫描到该设备的时间 |
| 在线状态 | online / offline |
| 在线时长 | 当前连续在线时长 |
| 离线次数 | 设备离线次数统计 |
| 设备类型 | PC、开发板、手机、路由器、打印机、未知设备等 |
| 设备标签 | 用户手动标记，例如“RK3588 开发板” |
| 所属用户 | 设备负责人 |
| 所属部门 | 研发、测试、生产、运维等 |
| 风险等级 | normal / warning / critical |

---

### 2.3 设备在线时间统计

系统需要记录设备上下线事件。

示例：

```text
设备 A：
2026-06-05 09:00:00 online
2026-06-05 12:30:00 offline
2026-06-05 13:10:00 online
```

通过这些事件可以计算：

- 今日在线时长
- 本周在线时长
- 本月在线时长
- 最近一次上线时间
- 最近一次离线时间
- 在线率
- 离线次数

---

### 2.4 实时流量折线图

当用户在 Web 页面点击某个在线设备时，系统展示该设备的实时流量图。

展示指标：

| 指标 | 说明 |
|---|---|
| 入站流量 | RX bytes / RX rate |
| 出站流量 | TX bytes / TX rate |
| 总流量 | RX + TX |
| 当前速率 | bytes/s、KB/s、MB/s |
| 峰值速率 | 当前时间窗口内最大流量 |
| 平均速率 | 当前时间窗口内平均流量 |
| 流量趋势 | 折线图展示 |

前端图表推荐：

| 图表库 | 说明 |
|---|---|
| ECharts | 中文生态好，折线图、仪表盘、排行榜都很方便 |
| Chart.js | 简单轻量 |
| AntV G2 | 更适合复杂数据分析场景 |

推荐使用：

```text
ECharts
```

---

## 3. 系统总体架构

### 3.1 总体架构图

```text
+-------------------------------------------------------------+
|                         Web 前端                             |
|  Vue / React / Svelte + ECharts + WebSocket/SSE              |
+-----------------------------+-------------------------------+
                              |
                              | HTTP / WebSocket
                              v
+-------------------------------------------------------------+
|                         Go Web 后端                          |
| Gin / Fiber / Echo                                            |
|                                                             |
|  +-------------------+   +-------------------+              |
|  | 用户权限模块       |   | 设备管理模块       |              |
|  +-------------------+   +-------------------+              |
|  | 扫描任务模块       |   | 流量采集模块       |              |
|  +-------------------+   +-------------------+              |
|  | 告警模块           |   | 日志审计模块       |              |
|  +-------------------+   +-------------------+              |
|  | WebSocket 推送模块 |   | API 网关模块       |              |
|  +-------------------+   +-------------------+              |
+-----------------------------+-------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                         数据存储层                            |
|  PostgreSQL / MySQL + Redis + InfluxDB/Prometheus             |
+-----------------------------+-------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                         网络采集层                            |
|  ARP Scanner / Ping Scanner / TCP Probe / Agent / SNMP        |
+-------------------------------------------------------------+
```

---

### 3.2 推荐技术栈

| 层级 | 技术选型 | 说明 |
|---|---|---|
| 后端语言 | Go | 高并发、部署简单、适合网络扫描和 Web 服务 |
| Web 框架 | Gin / Fiber | Gin 稳定成熟，Fiber 性能高 |
| 数据库 | PostgreSQL | 适合资产、用户、权限、日志等关系数据 |
| 缓存 | Redis | 存储实时在线状态、扫描任务队列、会话缓存 |
| 时序数据库 | InfluxDB / Prometheus | 存储流量、在线率、CPU、内存等时间序列数据 |
| 前端框架 | Vue 3 / React | 推荐 Vue 3，开发效率高 |
| UI 框架 | Element Plus / Ant Design Vue | 后台管理系统常用 |
| 图表 | ECharts | 实时折线图、排行榜、仪表盘 |
| 实时推送 | WebSocket / SSE | 设备状态和流量实时推送 |
| 部署 | Docker Compose | 初期部署简单 |
| 网关 | Nginx | 反向代理、HTTPS、静态资源 |
| 日志 | Zap / Zerolog | Go 高性能日志库 |
| 配置 | Viper | 配置文件管理 |
| ORM | GORM / Ent | 数据库操作 |

推荐组合：

```text
Go + Gin + PostgreSQL + Redis + InfluxDB + Vue3 + Element Plus + ECharts + WebSocket
```

---

## 4. 后端服务模块设计

### 4.1 模块划分

```text
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/              # HTTP API 路由
│   ├── auth/             # 登录认证、JWT、权限
│   ├── user/             # 用户管理
│   ├── device/           # 设备管理
│   ├── scanner/          # ARP/Ping/TCP 扫描
│   ├── traffic/          # 流量采集
│   ├── metrics/          # 指标管理
│   ├── alert/            # 告警模块
│   ├── websocket/        # 实时推送
│   ├── audit/            # 操作审计
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接
│   └── scheduler/        # 周期任务调度
├── pkg/
│   ├── netutil/          # 网络工具函数
│   ├── logger/           # 日志封装
│   └── response/         # API 响应结构
├── configs/
│   └── config.yaml
├── deployments/
│   ├── docker-compose.yml
│   └── nginx.conf
└── go.mod
```

---

### 4.2 服务启动流程

```text
1. 读取配置文件
2. 初始化日志系统
3. 连接 PostgreSQL
4. 连接 Redis
5. 连接 InfluxDB 或 Prometheus
6. 初始化数据库表结构
7. 启动周期扫描调度器
8. 启动流量采集任务
9. 启动 WebSocket Hub
10. 启动 HTTP Web 服务
```

Go 伪代码：

```go
func main() {
    cfg := config.Load()
    logger.Init(cfg.Log)

    db := database.InitPostgres(cfg.Database)
    redis := database.InitRedis(cfg.Redis)
    tsdb := database.InitInfluxDB(cfg.InfluxDB)

    scannerSvc := scanner.NewScannerService(db, redis)
    trafficSvc := traffic.NewTrafficService(db, tsdb)
    wsHub := websocket.NewHub()

    scheduler.StartDeviceScan(scannerSvc)
    scheduler.StartTrafficCollector(trafficSvc)

    router := api.NewRouter(db, redis, scannerSvc, trafficSvc, wsHub)
    router.Run(cfg.Server.Addr)
}
```

---

## 5. 网络扫描模块设计

### 5.1 扫描策略

推荐采用多阶段扫描：

```text
第一阶段：获取本机网卡和网段
第二阶段：生成扫描 IP 列表
第三阶段：ARP 扫描
第四阶段：Ping 扫描补充
第五阶段：TCP 端口探测补充
第六阶段：结果合并去重
第七阶段：更新设备状态
第八阶段：写入设备上下线事件
```

---

### 5.2 当前网段识别

系统启动后先识别本机网卡信息：

```text
网卡名称：eth0
本机 IP：192.168.1.10
子网掩码：255.255.255.0
网段：192.168.1.0/24
网关：192.168.1.1
```

Go 可以使用：

```go
net.Interfaces()
net.InterfaceAddrs()
```

获取网段后自动生成 IP 列表：

```text
192.168.1.1
192.168.1.2
192.168.1.3
...
192.168.1.254
```

---

### 5.3 ARP 扫描

ARP 扫描适合同一局域网内设备发现。

工作方式：

```text
向目标 IP 发送 ARP Request
如果设备在线，则返回 ARP Reply
ARP Reply 中包含 IP 与 MAC 地址
```

Go 实现可选方案：

| 方案 | 说明 |
|---|---|
| mdlayher/arp | Go 原生 ARP 库 |
| google/gopacket | 支持更底层的数据包构造 |
| 调用 arp-scan 命令 | 实现简单，但依赖系统命令 |

推荐开发阶段先调用系统命令：

```bash
arp-scan --localnet
```

后期再替换为 Go 原生实现。

---

### 5.4 Ping 扫描

Ping 用于补充判断设备是否在线。

Go 可使用：

| 库 | 说明 |
|---|---|
| go-ping/ping | 常用 ICMP ping 库 |
| os/exec 调用 ping | 简单但依赖系统命令 |

注意：

有些设备会禁用 ICMP，因此 ping 不通不代表设备一定离线。

---

### 5.5 TCP 探测

对于禁用 ICMP 的设备，可以扫描常见端口：

| 端口 | 服务 |
|---|---|
| 22 | SSH |
| 80 | HTTP |
| 443 | HTTPS |
| 445 | SMB |
| 3389 | RDP |
| 8080 | Web 服务 |
| 5000/8000/9000 | 常见嵌入式服务 |

TCP 探测伪代码：

```go
func TCPProbe(ip string, ports []int, timeout time.Duration) bool {
    for _, port := range ports {
        addr := fmt.Sprintf("%s:%d", ip, port)
        conn, err := net.DialTimeout("tcp", addr, timeout)
        if err == nil {
            conn.Close()
            return true
        }
    }
    return false
}
```

---

### 5.6 扫描并发模型

Go 非常适合并发扫描。

推荐使用 Worker Pool：

```text
IP 列表 → Job Channel → N 个 Worker 并发扫描 → Result Channel → 聚合结果
```

示意图：

```text
+-----------+      +-------------+      +-------------+
| IP 列表    | ---> | Job Channel | ---> | Worker Pool |
+-----------+      +-------------+      +-------------+
                                      |      |      |
                                      v      v      v
                                  Scan   Scan   Scan
                                      |      |      |
                                      v      v      v
                                +-------------------+
                                | Result Aggregator |
                                +-------------------+
```

推荐参数：

| 参数 | 建议值 |
|---|---|
| 小型局域网 /24 | 50 ~ 100 workers |
| 中型局域网 /22 | 100 ~ 300 workers |
| 大型局域网 | 按网段分批扫描 |
| 单 IP 超时 | 300ms ~ 1000ms |
| 扫描周期 | 30s ~ 300s |

---

## 6. 设备状态管理

### 6.1 状态模型

设备状态建议分为：

```text
unknown   未知
online    在线
offline   离线
warning   异常
blocked   已屏蔽
ignored   已忽略
```

---

### 6.2 在线状态判断逻辑

不要因为一次扫描失败就立即判断离线。

推荐规则：

```text
连续 1 次扫描成功：online
连续 3 次扫描失败：offline
连续多次上线/离线抖动：warning
```

这样可以避免网络抖动导致误判。

---

### 6.3 上下线事件

每次状态发生变化时，写入事件表。

示例：

```text
device_id = 1001
old_status = offline
new_status = online
event_time = 2026-06-05 09:00:00
```

用途：

- 计算在线时长
- 绘制在线时间轴
- 分析设备稳定性
- 生成告警

---

## 7. 流量采集模块设计

### 7.1 流量采集难点

仅通过 ARP 或 Ping 无法直接知道某个设备的实时流量。

要统计每个设备的流量，需要系统位于关键网络路径上，或者从交换机、路由器、防火墙、Agent 获取数据。

可选方案：

| 方案 | 能否获取单设备流量 | 难度 | 说明 |
|---|---:|---:|---|
| 本机网卡统计 | 否 | 低 | 只能看到本机整体流量 |
| 路由器 API | 是 | 中 | 依赖路由器支持 |
| SNMP 交换机 | 部分可以 | 中 | 可统计端口流量，需交换机支持 |
| NetFlow/sFlow | 是 | 高 | 企业网络常用方案 |
| eBPF 抓包 | 是 | 高 | 需要旁路或网关位置 |
| Agent 上报 | 是 | 中 | 每台设备安装 Agent |
| 镜像端口抓包 | 是 | 高 | 需要交换机端口镜像 |

因此推荐分阶段实现。

---

### 7.2 初期流量方案

如果系统只是部署在普通局域网内的一台机器上，建议初期实现：

```text
1. 记录设备在线状态
2. 记录设备上下线时间
3. 记录本机整体网卡流量
4. 对安装 Agent 的设备记录单设备流量
```

Agent 设备上报格式：

```json
{
  "device_id": "rk3588-001",
  "ip": "192.168.1.50",
  "mac": "00:11:22:33:44:55",
  "rx_bytes": 1024000,
  "tx_bytes": 2048000,
  "cpu_usage": 25.6,
  "memory_usage": 58.2,
  "timestamp": 1780650000
}
```

---

### 7.3 中期流量方案

中期可以接入：

```text
SNMP 交换机 + 路由器 API + Agent
```

可实现：

- 交换机端口流量
- 路由器出口流量
- 指定设备流量
- 设备流量排行榜
- 异常流量告警

---

### 7.4 高级流量方案

高级版本可以使用：

```text
NetFlow / sFlow / IPFIX + eBPF + DPI 元数据分析
```

可实现：

- 每台设备实时上下行流量
- 访问外网目标统计
- 流量排行榜
- 大流量设备检测
- 异常连接检测
- AI 工具访问量统计
- 外部网络连接排序

注意：

涉及网络流量采集时，必须确保系统部署在授权网络中，并遵守公司内部安全规范和隐私要求。

---

## 8. 前端 Web 页面设计

### 8.1 页面结构

```text
Web 管理后台
├── 登录页
├── 首页 Dashboard
├── 在线设备检测
│   ├── 在线设备列表
│   ├── 设备详情
│   ├── 在线时间折线图
│   ├── 实时流量折线图
│   └── 设备上下线事件
├── 设备资产管理
│   ├── 设备列表
│   ├── 新设备标记
│   ├── 设备分组
│   └── 设备标签
├── 流量分析
│   ├── 实时流量排行
│   ├── 外网访问排行
│   ├── 大流量设备排行
│   └── 异常流量分析
├── 告警中心
│   ├── 新设备告警
│   ├── 离线告警
│   ├── 大流量告警
│   └── 风险设备告警
├── 用户管理
│   ├── 用户列表
│   ├── 角色管理
│   ├── 权限管理
│   └── 登录日志
├── 系统设置
│   ├── 扫描网段配置
│   ├── 扫描周期配置
│   ├── 告警规则配置
│   └── 数据保留策略
└── 操作审计
```

---

### 8.2 Dashboard 首页

首页展示：

| 指标 | 说明 |
|---|---|
| 当前在线设备数 | 当前在线设备数量 |
| 今日新增设备数 | 今天首次发现的设备 |
| 今日离线设备数 | 今天离线过的设备 |
| 总设备数 | 系统累计发现设备 |
| 当前总流量 | 当前局域网或系统观测到的流量 |
| 高风险设备数 | 被标记为异常的设备 |
| 最近告警 | 最新告警列表 |

---

### 8.3 在线设备检测页面

设备列表字段：

| 字段 | 说明 |
|---|---|
| 状态 | 在线 / 离线 |
| IP | 设备 IP 地址 |
| MAC | 设备 MAC 地址 |
| 主机名 | 设备名称 |
| 厂商 | MAC OUI 厂商 |
| 在线时长 | 当前连续在线时间 |
| 当前流量 | 当前设备流量 |
| 标签 | 用户自定义标签 |
| 最后发现时间 | 最近一次扫描到设备时间 |
| 操作 | 查看详情、标记、忽略、告警配置 |

---

### 8.4 设备详情页面

点击在线设备后进入详情页。

页面内容：

```text
设备基础信息
在线状态卡片
在线时间折线图
实时流量折线图
设备上下线事件表
设备标签与备注
告警历史
操作日志
```

折线图示例：

```text
X 轴：时间
Y 轴：流量速率 / 在线状态
曲线 1：RX
曲线 2：TX
曲线 3：总流量
```

---

## 9. 用户与权限管理

### 9.1 用户角色

建议设计三类角色：

| 角色 | 权限 |
|---|---|
| super_admin | 系统全部权限 |
| admin | 设备管理、告警管理、用户查看 |
| operator | 查看设备、查看图表、处理告警 |
| viewer | 只读查看 |

---

### 9.2 权限模型

采用 RBAC：

```text
User 用户
Role 角色
Permission 权限
```

权限示例：

```text
device:view
device:update
device:delete
scan:start
scan:config
traffic:view
alert:view
alert:update
user:create
user:update
user:delete
audit:view
```

---

### 9.3 登录认证

推荐方式：

```text
用户名 + 密码 + JWT + Refresh Token
```

可选增强：

- 密码加盐哈希存储
- 登录失败次数限制
- IP 黑名单
- 二次认证
- 操作审计

---

## 10. 数据库设计

### 10.1 关系数据库表

推荐使用 PostgreSQL。

#### users 用户表

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(128),
    role_id BIGINT,
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### roles 角色表

```sql
CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### permissions 权限表

```sql
CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(128) UNIQUE NOT NULL,
    description TEXT
);
```

#### devices 设备表

```sql
CREATE TABLE devices (
    id BIGSERIAL PRIMARY KEY,
    ip VARCHAR(64) NOT NULL,
    mac VARCHAR(64),
    hostname VARCHAR(128),
    vendor VARCHAR(128),
    device_type VARCHAR(64) DEFAULT 'unknown',
    label VARCHAR(128),
    owner VARCHAR(128),
    department VARCHAR(128),
    status VARCHAR(32) DEFAULT 'unknown',
    first_seen TIMESTAMP,
    last_seen TIMESTAMP,
    last_online_at TIMESTAMP,
    last_offline_at TIMESTAMP,
    online_duration_seconds BIGINT DEFAULT 0,
    offline_count INT DEFAULT 0,
    fail_count INT DEFAULT 0,
    risk_level VARCHAR(32) DEFAULT 'normal',
    note TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(mac)
);
```

#### device_events 设备事件表

```sql
CREATE TABLE device_events (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    old_status VARCHAR(32),
    new_status VARCHAR(32),
    event_time TIMESTAMP DEFAULT NOW(),
    description TEXT
);
```

#### scan_tasks 扫描任务表

```sql
CREATE TABLE scan_tasks (
    id BIGSERIAL PRIMARY KEY,
    subnet VARCHAR(64) NOT NULL,
    scan_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    total_ips INT DEFAULT 0,
    online_count INT DEFAULT 0,
    error_message TEXT
);
```

#### alerts 告警表

```sql
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT,
    alert_type VARCHAR(64) NOT NULL,
    level VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    status VARCHAR(32) DEFAULT 'open',
    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP
);
```

#### audit_logs 审计日志表

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    action VARCHAR(128) NOT NULL,
    resource_type VARCHAR(64),
    resource_id VARCHAR(64),
    ip VARCHAR(64),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

### 10.2 时序数据设计

流量数据适合存入 InfluxDB 或 Prometheus。

Measurement：`device_traffic`

字段：

```text
device_id
ip
mac
rx_bytes
tx_bytes
rx_rate
tx_rate
total_rate
timestamp
```

InfluxDB 示例：

```text
device_traffic,device_id=1001,ip=192.168.1.50 rx_rate=1024,tx_rate=2048,total_rate=3072 1780650000000000000
```

---

## 11. API 接口设计

### 11.1 认证接口

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/logout | 登出 |
| POST | /api/v1/auth/refresh | 刷新 Token |
| GET | /api/v1/auth/profile | 当前用户信息 |

---

### 11.2 设备接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/devices | 获取设备列表 |
| GET | /api/v1/devices/online | 获取在线设备 |
| GET | /api/v1/devices/:id | 获取设备详情 |
| PUT | /api/v1/devices/:id | 更新设备信息 |
| DELETE | /api/v1/devices/:id | 删除设备 |
| POST | /api/v1/devices/:id/ignore | 忽略设备 |
| POST | /api/v1/devices/:id/label | 标记设备 |

---

### 11.3 扫描接口

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/scans/start | 手动开始扫描 |
| GET | /api/v1/scans/tasks | 获取扫描任务 |
| GET | /api/v1/scans/tasks/:id | 获取扫描任务详情 |
| PUT | /api/v1/scans/config | 修改扫描配置 |
| GET | /api/v1/scans/config | 获取扫描配置 |

---

### 11.4 流量接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/traffic/devices/:id/realtime | 获取设备实时流量 |
| GET | /api/v1/traffic/devices/:id/history | 获取设备历史流量 |
| GET | /api/v1/traffic/rank | 获取流量排行榜 |
| GET | /api/v1/traffic/summary | 获取流量汇总 |

---

### 11.5 WebSocket 接口

```text
/ws/devices/status
/ws/devices/:id/traffic
/ws/alerts
```

推送数据示例：

```json
{
  "type": "device_status",
  "data": {
    "device_id": 1001,
    "ip": "192.168.1.50",
    "status": "online",
    "timestamp": 1780650000
  }
}
```

---

## 12. 实时图表数据流

### 12.1 实时流量数据流

```text
设备 / Agent / SNMP / Router
        |
        v
Traffic Collector
        |
        v
InfluxDB / Prometheus
        |
        v
Go Web Backend
        |
        v
WebSocket / SSE
        |
        v
ECharts 折线图
```

---

### 12.2 在线状态数据流

```text
ARP / Ping / TCP Scanner
        |
        v
Scanner Service
        |
        v
Device Status Manager
        |
        v
PostgreSQL + Redis
        |
        v
WebSocket 推送
        |
        v
在线设备页面实时刷新
```

---

## 13. 配置文件设计

`config.yaml` 示例：

```yaml
server:
  addr: ":8080"
  mode: "release"

scan:
  enabled: true
  interval_seconds: 60
  subnets:
    - "192.168.1.0/24"
    - "192.168.2.0/24"
  methods:
    - "arp"
    - "ping"
    - "tcp"
  worker_count: 100
  timeout_ms: 800
  offline_threshold: 3
  tcp_ports:
    - 22
    - 80
    - 443
    - 8080

database:
  host: "postgres"
  port: 5432
  user: "lan_admin"
  password: "lan_password"
  dbname: "lan_monitor"

redis:
  addr: "redis:6379"
  password: ""
  db: 0

influxdb:
  url: "http://influxdb:8086"
  token: "change-me"
  org: "lan"
  bucket: "metrics"

jwt:
  secret: "change-this-secret"
  access_token_expire_minutes: 60
  refresh_token_expire_hours: 168

alert:
  new_device_enabled: true
  offline_enabled: true
  high_traffic_enabled: true
  high_traffic_threshold_mbps: 100
```

---

## 14. Docker Compose 部署设计

```yaml
version: "3.8"

services:
  backend:
    image: lan-monitor-backend:latest
    container_name: lan-monitor-backend
    network_mode: host
    privileged: true
    volumes:
      - ./configs:/app/configs
    depends_on:
      - postgres
      - redis
      - influxdb

  frontend:
    image: lan-monitor-frontend:latest
    container_name: lan-monitor-frontend
    ports:
      - "8081:80"
    depends_on:
      - backend

  postgres:
    image: postgres:16
    container_name: lan-monitor-postgres
    environment:
      POSTGRES_USER: lan_admin
      POSTGRES_PASSWORD: lan_password
      POSTGRES_DB: lan_monitor
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7
    container_name: lan-monitor-redis
    ports:
      - "6379:6379"

  influxdb:
    image: influxdb:2
    container_name: lan-monitor-influxdb
    ports:
      - "8086:8086"
    volumes:
      - influxdb_data:/var/lib/influxdb2

volumes:
  postgres_data:
  influxdb_data:
```

注意：

如果后端需要执行 ARP 扫描、抓包、读取网卡信息，Docker 容器通常需要：

```text
network_mode: host
privileged: true
CAP_NET_RAW
CAP_NET_ADMIN
```

生产环境中建议尽量减少 privileged 权限，只授予必要 Capability。

---

## 15. 安全设计

### 15.1 网络安全

- 仅允许在授权局域网内部署
- 限制扫描网段白名单
- 限制扫描频率，避免影响网络
- 不做漏洞利用，不做密码爆破
- 不主动攻击设备
- 对敏感接口做权限控制

---

### 15.2 Web 安全

- JWT 鉴权
- 密码哈希存储，推荐 bcrypt / argon2
- HTTPS 部署
- API 权限校验
- CSRF / XSS 防护
- 登录失败限制
- 操作审计
- 管理员操作二次确认

---

### 15.3 数据安全

- 数据库密码不写死在代码中
- 配置文件权限控制
- 定期备份数据库
- 日志脱敏
- MAC、IP、主机名等资产信息限制访问

---

## 16. 告警规则设计

### 16.1 新设备告警

条件：

```text
扫描到一个从未出现过的 MAC 地址
```

动作：

```text
生成告警
Web 页面弹窗
可选邮件/企业微信/钉钉通知
```

---

### 16.2 设备离线告警

条件：

```text
核心设备连续 N 次扫描失败
```

动作：

```text
告警等级 warning / critical
记录离线时间
通知负责人
```

---

### 16.3 大流量告警

条件：

```text
设备流量超过阈值，例如 100 Mbps 持续 5 分钟
```

动作：

```text
生成告警
加入流量排行榜
标记为 high_traffic
```

---

### 16.4 异常上下线告警

条件：

```text
设备在短时间内频繁上下线
```

适合发现：

- 网络不稳定设备
- 电源不稳定设备
- WiFi 信号差设备
- 系统反复重启的嵌入式设备

---

## 17. 设备 Agent 设计

如果需要更准确的设备状态和流量数据，建议开发轻量级 Agent。

### 17.1 Agent 功能

| 功能 | 说明 |
|---|---|
| 设备注册 | 首次启动向服务端注册 |
| 心跳上报 | 定期上报在线状态 |
| 流量上报 | 上报网卡 RX/TX |
| 系统信息 | CPU、内存、磁盘、温度 |
| 进程状态 | 可选，监控关键进程 |
| 远程命令 | 可选，需严格权限控制 |

---

### 17.2 Agent 上报接口

```text
POST /api/v1/agents/register
POST /api/v1/agents/heartbeat
POST /api/v1/agents/metrics
```

---

### 17.3 Agent 部署场景

适合安装在：

- RK3568 / RK3576 / RK3588 开发板
- Linux 工控机
- 医疗设备主机
- 测试服务器
- 关键网络节点

---

## 18. 开发阶段划分

### 第一阶段：MVP 版本

目标：先做出可用系统。

功能：

```text
用户登录
设备扫描
在线设备列表
设备详情
设备上下线记录
基础 Dashboard
手动扫描
周期扫描
```

技术：

```text
Go + Gin + PostgreSQL + Vue3 + ECharts
```

---

### 第二阶段：实时可视化版本

功能：

```text
WebSocket 实时推送
在线状态实时刷新
在线时间折线图
基础流量折线图
Redis 缓存
告警中心
```

---

### 第三阶段：Agent 版本

功能：

```text
Linux Agent
设备主动注册
CPU/内存/磁盘/网卡流量上报
设备健康度评分
设备标签管理
```

---

### 第四阶段：企业增强版

功能：

```text
SNMP 交换机接入
NetFlow/sFlow 接入
多网段扫描
多租户
权限细化
邮件/企业微信/钉钉告警
数据报表
```

---

## 19. 推荐项目目录

```text
lan-monitor/
├── backend/
│   ├── cmd/
│   ├── internal/
│   ├── pkg/
│   ├── configs/
│   ├── migrations/
│   └── go.mod
├── frontend/
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── vite.config.ts
├── agent/
│   ├── cmd/
│   ├── internal/
│   └── go.mod
├── docs/
│   ├── architecture.md
│   ├── api.md
│   └── deployment.md
├── deployments/
│   ├── docker-compose.yml
│   ├── nginx.conf
│   └── systemd/
└── README.md
```

---

## 20. 关键技术点学习路线

### Go 后端方向

- Go 基础语法
- Goroutine / Channel
- Context 超时控制
- Worker Pool 并发模型
- Gin / Fiber Web 框架
- GORM / SQLX
- JWT 鉴权
- WebSocket
- 配置管理 Viper
- 日志 Zap

### 网络方向

- ARP 协议
- ICMP Ping
- TCP 连接探测
- MAC 地址与 OUI
- 子网掩码与 CIDR
- SNMP
- NetFlow / sFlow
- Linux 网卡流量统计
- eBPF 基础

### 前端方向

- Vue 3
- TypeScript
- Element Plus
- ECharts
- WebSocket 实时图表
- 后台管理系统布局

### 运维方向

- Docker
- Docker Compose
- PostgreSQL
- Redis
- InfluxDB / Prometheus
- Nginx
- systemd
- 日志与监控

---

## 21. MVP 实现优先级

建议先按以下顺序开发：

```text
1. 后端项目初始化
2. PostgreSQL 表结构
3. 用户登录与 JWT
4. 扫描配置读取
5. 获取本机网段
6. Ping 扫描
7. ARP 表读取
8. 设备入库
9. 在线状态判断
10. 在线设备列表 API
11. 前端登录页
12. 前端设备列表页
13. 设备详情页
14. 上下线事件记录
15. ECharts 在线时间图
16. WebSocket 推送
17. Agent 流量上报
18. ECharts 实时流量图
```

---

## 22. 需要特别注意的问题

### 22.1 ARP 扫描只能发现同一二层网络

如果公司内部有多个 VLAN 或多个网段，仅靠 ARP 扫描无法跨网段发现设备。

解决方案：

```text
1. 每个网段部署一个采集节点
2. 接入路由器/DHCP/交换机数据
3. 使用 Agent 主动注册
4. 通过多网段配置进行 Ping/TCP 探测
```

---

### 22.2 单设备流量不能只靠 Ping 获取

Ping 只能判断设备是否在线，不能获取设备真实流量。

要获得真实流量，必须使用：

```text
Agent / SNMP / NetFlow / 路由器 API / 镜像端口 / 网关抓包
```

---

### 22.3 扫描频率不能太高

过高频率可能影响网络设备或被安全系统误判。

建议：

```text
普通设备：60 秒扫描一次
核心设备：10~30 秒心跳检测
大网段：分批扫描
```

---

### 22.4 MAC 地址可能变化

某些手机、电脑开启随机 MAC 后，MAC 地址会变化。

解决方案：

```text
IP + MAC + Hostname + Agent ID 综合识别
```

---

## 23. 最终推荐架构

对于你的需求，推荐最终架构如下：

```text
Go Backend:
  Gin + GORM + JWT + WebSocket + Worker Pool Scanner

Database:
  PostgreSQL 存设备、用户、事件、告警
  Redis 存实时状态和缓存
  InfluxDB 存流量和时序指标

Scanner:
  ARP + Ping + TCP Probe

Traffic:
  初期：Agent 上报 + 本机网卡统计
  中期：SNMP / 路由器 API
  后期：NetFlow / sFlow / eBPF

Frontend:
  Vue3 + Element Plus + ECharts

Deployment:
  Docker Compose + Nginx
```

---

## 24. 总结

这个系统可以分为五个核心层：

```text
1. 网络发现层：ARP / Ping / TCP / Agent
2. 数据采集层：设备状态、在线时间、流量、系统指标
3. 后端服务层：Go Web API、扫描调度、权限、告警
4. 数据存储层：PostgreSQL、Redis、InfluxDB
5. 前端展示层：设备列表、详情页、实时折线图、排行榜
```

最重要的设计原则是：

```text
先实现在线设备发现，再实现设备状态管理；
先实现基础图表，再引入真实流量采集；
先做单网段 MVP，再扩展多网段、Agent、SNMP 和 NetFlow。
```

这样系统可以从一个简单可用的局域网在线设备检测工具，逐步演进为一个完整的公司内部网络资产与设备运维管理平台。
