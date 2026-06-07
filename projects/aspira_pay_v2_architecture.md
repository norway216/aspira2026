# Aspira Pay V2 跨境支付清算交易系统架构设计

> 版本：V2 Sandbox / 准生产级架构  
> 作者：Aspira Studio  
> 技术方向：Go + C++ + 区块链清算账本 + 高并发交易系统  
> 文档目标：设计一个高性能、高稳定性、低延迟、低资源占用、易部署、可审计、可追溯的跨境支付清算交易系统。

---

## 1. 系统定位

Aspira Pay V2 是一个面向跨境支付、清算、审计和链上追溯的交易系统。它不是简单的转账接口，而是一个完整的金融交易基础设施。

系统核心目标：

1. 使用 Go 实现支付业务流程、KYC、风控、订单状态机、清算编排、链上适配。
2. 使用 C++ 实现高性能交易清算引擎，处理账户冻结、扣减、入账、事件生成。
3. 使用区块链技术记录交易过程中的关键状态，实现不可篡改、可追溯、可审计。
4. 使用消息队列实现异步化、削峰、解耦和事件驱动。
5. 使用复式记账保证资金账本严谨。
6. 使用 Docker Compose / Kubernetes 实现易部署和可扩展。
7. 支持高并发、高性能、低延迟、低资源占用。

---

## 2. V2 架构边界

V2 层级建议定位为 **Sandbox / 准生产环境**。

### 2.1 V2 支持的能力

| 能力 | 是否支持 | 说明 |
|---|---:|---|
| 用户注册 | 支持 | 可创建用户账户 |
| KYC 流程 | 支持 | 支持模拟 KYC、人工审核、风险等级 |
| AML 风控 | 支持 | 支持规则风控、黑名单、限额、国家限制 |
| 跨境支付订单 | 支持 | 支持多币种支付订单 |
| FX 汇率报价 | 支持 | 支持模拟汇率与报价锁定 |
| 资金冻结 | 支持 | C++ 引擎处理冻结逻辑 |
| 交易清算 | 支持 | C++ 引擎 + Go Settlement Service |
| 复式记账 | 支持 | PostgreSQL 账本分录 |
| 区块链审计 | 支持 | 联盟链 / 许可链记录交易哈希 |
| 管理后台 | 支持 | 查询交易、用户、账本、审计 |
| 实时监控 | 支持 | Prometheus + Grafana |
| 真实银行通道 | 暂不支持 | V2 不直接接真实银行 |
| 真实资金出入金 | 暂不支持 | 使用模拟账户体系 |
| 真实外汇撮合 | 暂不支持 | 使用模拟 FX Quote |

### 2.2 V2 不做的事情

V2 不直接处理真实客户资金，不接真实银行通道，不做真实外汇交易，不绕过任何金融监管。

V2 的目标是完成：

```text
用户身份认证
    ↓
交易风控
    ↓
支付订单创建
    ↓
资金冻结
    ↓
C++ 高性能交易引擎处理
    ↓
Go 清算服务落账
    ↓
区块链记录交易状态
    ↓
管理后台查询与审计
```

---

## 3. 总体架构

### 3.1 总体架构图

```text
┌──────────────────────────────────────────────────────────────────┐
│                        Web Admin / Dashboard                     │
│        React / Vue / Svelte + TailwindCSS / Apple Music 风格       │
└───────────────────────────────┬──────────────────────────────────┘
                                │ HTTPS
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│           Go / JWT / mTLS / Rate Limit / Request Signature         │
└───────────────┬───────────────────────┬──────────────────────────┘
                │                       │
                ▼                       ▼
┌───────────────────────┐     ┌──────────────────────────────┐
│ User Service           │     │ KYC Service                   │
│ Go                     │     │ Go                            │
│ 用户、账户、权限        │     │ 身份认证、风险等级、审核        │
└───────────┬───────────┘     └──────────────┬───────────────┘
            │                                │
            ▼                                ▼
┌───────────────────────┐     ┌──────────────────────────────┐
│ Risk / AML Service     │     │ FX Quote Service              │
│ Go                     │     │ Go                            │
│ 规则风控、黑名单、限额   │     │ 汇率报价、报价锁定             │
└───────────┬───────────┘     └──────────────┬───────────────┘
            │                                │
            └──────────────┬─────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Payment Service                          │
│      Go / 支付订单状态机 / 幂等控制 / 交易编排 / Outbox Pattern     │
└───────────────────────────────┬──────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Message Queue                            │
│              NATS JetStream / Redpanda / Kafka                    │
│   payment.created / risk.checked / engine.command / engine.done   │
└───────────────────────────────┬──────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                     C++ Trading & Clearing Engine                 │
│     C++20 / Lock-Free Queue / In-Memory Ledger / WAL / Low Latency │
│     资金冻结 / 扣款 / 入账 / 手续费 / 清算事件 / 顺序执行           │
└───────────────────────────────┬──────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                       Settlement Service                         │
│       Go / 复式记账 / 清算批次 / 对账 / 失败补偿 / 事件重放         │
└───────────────────────────────┬──────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Blockchain Layer                         │
│        Hyperledger Fabric / Tendermint / CometBFT / Audit Chain    │
│        交易哈希 / 状态哈希 / 清算批次哈希 / 审计签名               │
└───────────────────────────────┬──────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Data Layer                               │
│ PostgreSQL / Redis / ClickHouse / MinIO / RocksDB / Object Storage│
└──────────────────────────────────────────────────────────────────┘
```

---

## 4. 系统核心模块

### 4.1 API Gateway

API Gateway 是系统入口。

职责：

1. 统一接收外部请求。
2. JWT 鉴权。
3. API 签名校验。
4. 请求限流。
5. IP 白名单。
6. 防重放攻击。
7. 请求日志与审计。
8. 路由到内部微服务。

推荐技术：

```text
Go + Gin / Fiber / Echo
JWT
mTLS
Redis Rate Limit
OpenTelemetry
```

接口示例：

```text
POST /api/v2/users/register
POST /api/v2/kyc/submit
POST /api/v2/payments
GET  /api/v2/payments/{payment_id}
GET  /api/v2/ledger/{payment_id}
GET  /api/v2/audit/{payment_id}
```

---

### 4.2 User Service

职责：

1. 用户注册。
2. 用户登录。
3. 账户状态管理。
4. 用户权限管理。
5. 用户风险等级绑定。
6. 用户冻结 / 解冻。

用户状态：

```go
type UserStatus string

const (
    UserPendingKYC UserStatus = "PENDING_KYC"
    UserActive     UserStatus = "ACTIVE"
    UserFrozen     UserStatus = "FROZEN"
    UserRejected   UserStatus = "REJECTED"
)
```

---

### 4.3 KYC Service

职责：

1. 身份资料采集。
2. 证件信息校验。
3. 人脸验证结果接入。
4. 地址验证。
5. KYC 状态机。
6. 人工审核。
7. KYC 审计记录。

KYC 状态：

```go
type KYCStatus string

const (
    KYCPending      KYCStatus = "PENDING"
    KYCReviewing   KYCStatus = "MANUAL_REVIEW"
    KYCApproved    KYCStatus = "APPROVED"
    KYCRejected    KYCStatus = "REJECTED"
    KYCExpired     KYCStatus = "EXPIRED"
)
```

KYC 风险等级：

```go
type KYCRiskLevel string

const (
    RiskLow    KYCRiskLevel = "LOW"
    RiskMedium KYCRiskLevel = "MEDIUM"
    RiskHigh   KYCRiskLevel = "HIGH"
)
```

---

### 4.4 Risk / AML Service

职责：

1. 交易前风控。
2. 用户黑名单检查。
3. 收款方黑名单检查。
4. 制裁名单模拟检查。
5. 国家 / 地区限制。
6. 单笔限额检查。
7. 日累计限额检查。
8. 高频交易检查。
9. 异常 IP / 设备指纹检查。
10. 人工审核队列。

风控结果：

```go
type RiskDecision string

const (
    RiskPass   RiskDecision = "PASS"
    RiskReject RiskDecision = "REJECT"
    RiskReview RiskDecision = "MANUAL_REVIEW"
)

type RiskResult struct {
    Decision RiskDecision `json:"decision"`
    Score    int          `json:"score"`
    Reasons  []string     `json:"reasons"`
}
```

规则示例：

```text
规则 1：用户未完成 KYC，拒绝交易
规则 2：单笔金额超过用户等级限额，进入人工审核
规则 3：收款国家在限制列表中，拒绝交易
规则 4：1 分钟内发起超过 10 笔交易，进入人工审核
规则 5：新用户 24 小时内大额交易，进入人工审核
```

---

### 4.5 FX Quote Service

职责：

1. 提供币种汇率。
2. 生成报价。
3. 锁定报价有效期。
4. 计算手续费。
5. 记录报价历史。

报价数据结构：

```go
type FXQuote struct {
    QuoteID        string `json:"quote_id"`
    SourceCurrency string `json:"source_currency"`
    TargetCurrency string `json:"target_currency"`
    Rate           string `json:"rate"`
    SourceAmount   int64  `json:"source_amount"`
    TargetAmount   int64  `json:"target_amount"`
    FeeAmount      int64  `json:"fee_amount"`
    ExpiresAt      int64  `json:"expires_at"`
}
```

注意：

```text
金额使用 int64 表示最小货币单位。
汇率使用 decimal / string / NUMERIC，不使用 float 或 double。
```

---

### 4.6 Payment Service

Payment Service 是 Go 业务编排核心。

职责：

1. 创建支付订单。
2. 幂等控制。
3. KYC 检查。
4. 风控检查。
5. 汇率报价锁定。
6. 订单状态机管理。
7. 投递交易命令到消息队列。
8. 接收交易结果事件。
9. 更新订单状态。

支付状态机：

```text
CREATED
  ↓
KYC_CHECKED
  ↓
RISK_CHECKED
  ↓
QUOTE_LOCKED
  ↓
FUNDS_FREEZE_REQUESTED
  ↓
SUBMITTED_TO_ENGINE
  ↓
ENGINE_EXECUTED
  ↓
SETTLEMENT_PENDING
  ↓
SETTLED
  ↓
CHAIN_CONFIRMED
  ↓
COMPLETED
```

异常状态：

```text
REJECTED
FAILED
CANCELLED
REFUNDED
MANUAL_REVIEW
```

状态定义：

```go
type PaymentStatus string

const (
    PaymentCreated              PaymentStatus = "CREATED"
    PaymentKYCChecked           PaymentStatus = "KYC_CHECKED"
    PaymentRiskChecked          PaymentStatus = "RISK_CHECKED"
    PaymentQuoteLocked          PaymentStatus = "QUOTE_LOCKED"
    PaymentFundsFreezeRequested PaymentStatus = "FUNDS_FREEZE_REQUESTED"
    PaymentSubmittedToEngine    PaymentStatus = "SUBMITTED_TO_ENGINE"
    PaymentEngineExecuted       PaymentStatus = "ENGINE_EXECUTED"
    PaymentSettlementPending    PaymentStatus = "SETTLEMENT_PENDING"
    PaymentSettled              PaymentStatus = "SETTLED"
    PaymentChainConfirmed       PaymentStatus = "CHAIN_CONFIRMED"
    PaymentCompleted            PaymentStatus = "COMPLETED"
    PaymentRejected             PaymentStatus = "REJECTED"
    PaymentFailed               PaymentStatus = "FAILED"
)
```

---

### 4.7 C++ Trading & Clearing Engine

C++ 引擎是系统性能核心。

在 Aspira Pay 中，它不是证券市场撮合引擎，而是高性能资金处理和清算引擎。

职责：

1. 接收交易命令。
2. 校验 sequence_id。
3. 校验 request_id 幂等。
4. 检查账户余额缓存。
5. 冻结付款方资金。
6. 扣款。
7. 计算手续费。
8. 写入 WAL。
9. 更新内存账本。
10. 生成清算事件。
11. 发布 engine.executed 事件。

#### 4.7.1 C++ 引擎内部架构

```text
┌────────────────────────────────────────────┐
│              Engine Adapter                │
│        gRPC / NATS / TCP / FlatBuffers      │
└──────────────────────┬─────────────────────┘
                       ▼
┌────────────────────────────────────────────┐
│              Command Decoder               │
│       校验签名 / request_id / sequence_id   │
└──────────────────────┬─────────────────────┘
                       ▼
┌────────────────────────────────────────────┐
│           Lock-Free Command Queue           │
│      Ring Buffer / MPSC Queue / Batch Pull  │
└──────────────────────┬─────────────────────┘
                       ▼
┌────────────────────────────────────────────┐
│              Core Engine Loop               │
│    Single Writer Principle / Ordered Exec   │
└──────────────────────┬─────────────────────┘
                       ▼
┌────────────────────────────────────────────┐
│              In-Memory Ledger               │
│     account_id -> available/frozen/settled  │
└──────────────────────┬─────────────────────┘
                       ▼
┌────────────────────────────────────────────┐
│                  WAL Log                    │
│       command log / event log / snapshot    │
└──────────────────────┬─────────────────────┘
                       ▼
┌────────────────────────────────────────────┐
│             Event Publisher                 │
│       NATS / Kafka / Redpanda / gRPC Stream │
└────────────────────────────────────────────┘
```

#### 4.7.2 C++ 引擎数据结构

```cpp
enum class CommandType {
    FREEZE_FUNDS,
    EXECUTE_PAYMENT,
    RELEASE_FUNDS,
    REFUND_PAYMENT,
    SETTLEMENT_BATCH
};

enum class EngineResult {
    ACCEPTED,
    REJECTED,
    EXECUTED,
    DUPLICATED,
    INSUFFICIENT_FUNDS
};

struct PaymentCommand {
    uint64_t sequence_id;
    std::string request_id;
    std::string payment_id;
    std::string from_account;
    std::string to_account;
    std::string source_currency;
    std::string target_currency;
    int64_t source_amount;
    int64_t target_amount;
    int64_t fee_amount;
    int64_t timestamp;
};

struct AccountBalance {
    int64_t available;
    int64_t frozen;
    int64_t settled;
};

struct EngineEvent {
    uint64_t sequence_id;
    std::string event_id;
    std::string payment_id;
    std::string event_type;
    std::string result;
    int64_t timestamp;
};
```

#### 4.7.3 C++ 引擎性能原则

```text
1. 热路径不直接写 PostgreSQL。
2. 热路径不做复杂 JSON 序列化。
3. 热路径不做远程 RPC 查询。
4. 金额使用 int64。
5. 尽量使用批处理。
6. 尽量减少锁。
7. 使用单写线程保证账本一致性。
8. 使用 WAL 保证崩溃恢复。
9. 使用 Snapshot 加快重启恢复。
10. 所有事件可重放。
```

---

### 4.8 Settlement Service

Settlement Service 负责将 C++ 引擎结果落到正式账本中。

职责：

1. 消费 engine.executed 事件。
2. 写入 ledger_entries。
3. 生成复式记账分录。
4. 创建 settlement_batch。
5. 对账。
6. 失败补偿。
7. 触发链上记录。
8. 更新支付订单状态。

复式记账原则：

```text
每一笔交易都必须借贷平衡。
账本只能追加，不能物理删除。
冲正必须生成反向分录。
每个 ledger_entry 必须有 event_id。
```

示例：

```text
用户 A 支付 100 USD，手续费 1 USD，用户 B 收到等值 JPY。

分录 1：
借：用户 A 可用余额 101 USD
贷：平台清算中间账户 101 USD

分录 2：
借：平台清算中间账户 100 USD
贷：用户 B 待入账余额 对应 JPY

分录 3：
借：平台清算中间账户 1 USD
贷：平台手续费收入账户 1 USD
```

---

### 4.9 Blockchain Layer

V2 的区块链层建议使用联盟链 / 许可链，不建议直接使用公链。

推荐方案：

```text
优先：Hyperledger Fabric
备选：Tendermint / CometBFT
学习版：PostgreSQL Append-only Ledger + Hash Chain
```

#### 4.9.1 链上记录什么

链上只记录关键摘要，不记录敏感明文数据。

| 数据 | 是否上链 | 说明 |
|---|---:|---|
| payment_id_hash | 是 | 支付订单 ID 哈希 |
| user_id_hash | 是 | 用户 ID 哈希 |
| amount_hash | 是 | 金额哈希 |
| status | 是 | 交易状态 |
| settlement_batch_id | 是 | 清算批次 |
| ledger_root_hash | 是 | 账本 Merkle Root |
| audit_signature | 是 | 审计签名 |
| KYC 明文 | 否 | 敏感数据不上链 |
| 身份证照片 | 否 | 敏感数据不上链 |
| 银行卡号 | 否 | 敏感数据不上链 |

链上记录结构：

```go
type ChainPaymentRecord struct {
    ChainTxID         string `json:"chain_tx_id"`
    PaymentIDHash     string `json:"payment_id_hash"`
    SenderHash        string `json:"sender_hash"`
    ReceiverHash      string `json:"receiver_hash"`
    AmountHash        string `json:"amount_hash"`
    CurrencyPair      string `json:"currency_pair"`
    Status            string `json:"status"`
    SettlementBatchID string `json:"settlement_batch_id"`
    LedgerRootHash    string `json:"ledger_root_hash"`
    Timestamp         int64  `json:"timestamp"`
    Signature         string `json:"signature"`
}
```

#### 4.9.2 链上状态流

```text
PAYMENT_CREATED
    ↓
RISK_APPROVED
    ↓
FUNDS_FROZEN
    ↓
ENGINE_EXECUTED
    ↓
SETTLED
    ↓
COMPLETED
```

#### 4.9.3 Hash Chain 结构

如果 V2 初期不直接上 Fabric，可以先做内部 Hash Chain：

```text
block_1_hash = hash(block_1_data + prev_hash)
block_2_hash = hash(block_2_data + block_1_hash)
block_3_hash = hash(block_3_data + block_2_hash)
```

表设计：

```sql
CREATE TABLE chain_blocks (
    id BIGSERIAL PRIMARY KEY,
    block_height BIGINT NOT NULL UNIQUE,
    block_hash VARCHAR(128) NOT NULL,
    prev_hash VARCHAR(128) NOT NULL,
    merkle_root VARCHAR(128) NOT NULL,
    event_count INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE chain_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    block_height BIGINT NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_hash VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

---

## 5. 数据层设计

### 5.1 PostgreSQL

PostgreSQL 负责核心业务数据：

```text
users
accounts
kyc_profiles
payment_orders
ledger_entries
settlement_batches
idempotency_keys
audit_logs
outbox_events
```

### 5.2 Redis

Redis 负责：

```text
短期幂等缓存
限流计数
KYC 状态缓存
用户风险等级缓存
热点账户余额只读缓存
短期报价缓存
```

注意：Redis 不是资金账本的最终来源，不能作为最终账本。

### 5.3 ClickHouse

ClickHouse 负责：

```text
交易分析
审计日志分析
风控统计
性能指标分析
交易排行榜
国家/币种维度统计
```

### 5.4 RocksDB / LMDB

C++ 引擎可以使用 RocksDB 或 LMDB 存储：

```text
WAL
Snapshot
Engine local state
Dedup request_id
```

### 5.5 MinIO

MinIO 用于存储：

```text
KYC 文件
证件照片
审计报表
对账文件
导出文件
```

敏感文件必须加密存储。

---

## 6. 核心数据库表设计

### 6.1 用户表

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    risk_level VARCHAR(32) NOT NULL DEFAULT 'LOW',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 6.2 账户表

```sql
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    available_balance BIGINT NOT NULL DEFAULT 0,
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    settled_balance BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'NORMAL',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(user_id, currency)
);
```

### 6.3 KYC 表

```sql
CREATE TABLE kyc_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    nationality VARCHAR(64),
    document_type VARCHAR(64),
    document_hash VARCHAR(255),
    address_hash VARCHAR(255),
    kyc_status VARCHAR(32) NOT NULL,
    risk_level VARCHAR(32) NOT NULL DEFAULT 'LOW',
    reviewed_by VARCHAR(64),
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 6.4 支付订单表

```sql
CREATE TABLE payment_orders (
    id BIGSERIAL PRIMARY KEY,
    payment_id VARCHAR(64) UNIQUE NOT NULL,
    request_id VARCHAR(128) UNIQUE NOT NULL,
    sender_user_id VARCHAR(64) NOT NULL,
    receiver_user_id VARCHAR(64) NOT NULL,
    source_currency VARCHAR(16) NOT NULL,
    target_currency VARCHAR(16) NOT NULL,
    source_amount BIGINT NOT NULL,
    target_amount BIGINT NOT NULL,
    fee_amount BIGINT NOT NULL,
    fx_rate NUMERIC(30, 12) NOT NULL,
    status VARCHAR(64) NOT NULL,
    risk_score INT DEFAULT 0,
    quote_id VARCHAR(64),
    chain_tx_id VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 6.5 账本分录表

```sql
CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    entry_id VARCHAR(64) UNIQUE NOT NULL,
    event_id VARCHAR(64) NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    direction VARCHAR(16) NOT NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 6.6 清算批次表

```sql
CREATE TABLE settlement_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) UNIQUE NOT NULL,
    currency VARCHAR(16) NOT NULL,
    total_debit BIGINT NOT NULL,
    total_credit BIGINT NOT NULL,
    entry_count INT NOT NULL,
    status VARCHAR(32) NOT NULL,
    ledger_root_hash VARCHAR(128),
    chain_tx_id VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 6.7 幂等表

```sql
CREATE TABLE idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) UNIQUE NOT NULL,
    request_hash VARCHAR(255) NOT NULL,
    response_body JSONB,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 6.8 Outbox 表

```sql
CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

---

## 7. 交易主流程

### 7.1 正常支付流程

```text
1. 用户提交支付请求
2. API Gateway 校验 JWT、签名、限流
3. Payment Service 检查 request_id 幂等
4. User Service 检查用户状态
5. KYC Service 检查 KYC 状态
6. Risk Service 执行 AML / 风控规则
7. FX Quote Service 生成报价并锁定
8. Payment Service 创建 payment_order
9. Payment Service 写 outbox_events
10. Outbox Worker 发布 payment.created
11. Engine Adapter 转发 engine.command
12. C++ Engine 执行资金冻结和扣减
13. C++ Engine 写 WAL
14. C++ Engine 发布 engine.executed
15. Settlement Service 消费 engine.executed
16. Settlement Service 写 ledger_entries
17. Settlement Service 生成 settlement_batch
18. Chain Service 写入区块链交易摘要
19. Payment Service 更新状态为 COMPLETED
20. Web Admin 可查询完整交易链路
```

### 7.2 时序图

```text
Client
  │
  │ POST /payments
  ▼
API Gateway
  │
  ▼
Payment Service
  │ check idempotency
  │ check user
  │ check kyc
  │ check risk
  │ lock quote
  │ create order
  ▼
Outbox Events
  │
  ▼
Message Queue
  │
  ▼
C++ Engine
  │ freeze funds
  │ execute payment
  │ write WAL
  ▼
Engine Event
  │
  ▼
Settlement Service
  │ write double-entry ledger
  │ create settlement batch
  ▼
Chain Service
  │ write payment hash to blockchain
  ▼
Payment Completed
```

---

## 8. 消息队列设计

### 8.1 推荐选型

| 组件 | 优点 | 使用建议 |
|---|---|---|
| NATS JetStream | 轻量、低资源、部署简单 | V2 首选 |
| Redpanda | Kafka 兼容、性能高 | 中型部署 |
| Kafka | 生态成熟、吞吐高 | 大型生产环境 |

V2 首选：

```text
NATS JetStream
```

原因：

```text
资源占用低
部署简单
延迟低
适合事件驱动
适合 Go 生态
```

### 8.2 Topic 设计

```text
payment.created
payment.kyc_checked
payment.risk_checked
payment.quote_locked
engine.command
engine.executed
engine.rejected
settlement.created
settlement.completed
chain.recorded
payment.completed
payment.failed
audit.event
```

### 8.3 事件结构

```go
type Event struct {
    EventID     string `json:"event_id"`
    EventType   string `json:"event_type"`
    AggregateID string `json:"aggregate_id"`
    PaymentID   string `json:"payment_id"`
    SequenceID  uint64 `json:"sequence_id"`
    PayloadHash string `json:"payload_hash"`
    CreatedAt   int64  `json:"created_at"`
}
```

---

## 9. 幂等与一致性设计

### 9.1 幂等原则

所有关键接口必须带：

```text
request_id
idempotency_key
payment_id
event_id
sequence_id
```

规则：

```text
相同 request_id + 相同 request_hash：返回上次结果
相同 request_id + 不同 request_hash：拒绝
相同 event_id：只消费一次
相同 engine sequence_id：只执行一次
```

### 9.2 Outbox Pattern

Payment Service 写订单和写事件必须在同一个数据库事务内完成：

```text
BEGIN
  INSERT payment_orders
  INSERT outbox_events
COMMIT
```

然后由 Outbox Worker 异步发布事件。

这样可以避免：

```text
订单创建成功，但消息发送失败
消息发送成功，但订单创建失败
```

### 9.3 Saga Pattern

跨服务事务使用 Saga：

```text
Create Payment
  ↓
Risk Check
  ↓
Freeze Funds
  ↓
Execute Payment
  ↓
Settlement
  ↓
Chain Record
```

失败补偿：

```text
Risk Reject -> Payment Rejected
Engine Failed -> Release Funds
Settlement Failed -> Retry / Manual Review
Chain Failed -> Retry Chain Record，不影响本地账本最终一致
```

---

## 10. 区块链交易过程设计

### 10.1 为什么使用区块链

Aspira Pay 使用区块链不是为了发行 Token，而是为了：

```text
交易状态不可篡改
多参与方清算共识
审计可追溯
清算批次可信证明
防止内部账本被篡改
跨机构对账
```

### 10.2 区块链不承担的职责

区块链不适合承担：

```text
高频热路径交易处理
KYC 明文存储
复杂业务查询
大文件存储
实时风控判断
```

正确做法：

```text
高频交易：C++ Engine
业务状态：PostgreSQL
审计证明：Blockchain
文件存储：MinIO
分析统计：ClickHouse
```

### 10.3 链上交易模型

每一个关键交易状态都生成一个链上事件：

```text
PaymentCreatedRecord
RiskApprovedRecord
FundsFrozenRecord
EngineExecutedRecord
SettlementCompletedRecord
PaymentCompletedRecord
```

链上记录示例：

```json
{
  "payment_id_hash": "sha256(payment_id)",
  "event_type": "SETTLEMENT_COMPLETED",
  "ledger_root_hash": "merkle_root_hash",
  "settlement_batch_id": "batch_20260607_0001",
  "timestamp": 1780848000,
  "signature": "ed25519_signature"
}
```

### 10.4 链上提交策略

为降低资源消耗，不建议每笔交易都立刻单独上链。

推荐：

```text
小额交易：批量上链
大额交易：单独上链
高风险交易：单独上链
普通状态：批量生成 Merkle Root 后上链
```

批量上链流程：

```text
1000 笔交易事件
    ↓
计算每个 event_hash
    ↓
生成 Merkle Tree
    ↓
得到 ledger_root_hash
    ↓
将 root_hash + batch_id 写入链
```

优点：

```text
减少链上写入次数
降低资源占用
提高吞吐
仍可证明单笔交易存在于批次中
```

---

## 11. 高并发与低延迟设计

### 11.1 高并发入口

API Gateway：

```text
连接池
协程池
限流
异步日志
请求签名快速校验
Redis 短缓存
```

Payment Service：

```text
无状态多副本
数据库连接池
读写分离
批量写 outbox
异步状态推进
```

### 11.2 C++ 引擎低延迟

C++ 引擎：

```text
MPSC Ring Buffer
Single Writer Principle
Batch Processing
Memory Ledger
WAL 顺序写
异步事件发布
CPU 亲和性绑定
减少动态内存分配
减少锁竞争
```

### 11.3 数据库优化

PostgreSQL：

```text
payment_id 建唯一索引
request_id 建唯一索引
created_at 分区
payment_orders 按月份分区
ledger_entries 按月份或 payment_id hash 分区
连接池 PgBouncer
批量插入 ledger_entries
```

### 11.4 链上写入优化

```text
批量上链
异步上链
Merkle Root 上链
失败重试
链上写入不阻塞主交易完成
```

交易主链路应该是：

```text
支付请求 -> 风控 -> 引擎执行 -> 本地账本完成
```

链上记录可以是最终一致：

```text
本地账本完成 -> 异步链上确认 -> 更新 chain_confirmed
```

---

## 12. 高可用设计

### 12.1 服务部署

```text
API Gateway：3 副本
Payment Service：3 副本
Risk Service：2 副本
KYC Service：2 副本
FX Service：2 副本
Settlement Service：2 副本
Chain Service：2 副本
C++ Engine：Active + Standby
PostgreSQL：Primary + Replica
Redis：Sentinel / Cluster
NATS：3 节点 JetStream
Blockchain：3 到 5 节点
```

### 12.2 C++ 引擎 HA

C++ 引擎采用：

```text
Active Engine + Standby Engine
```

同步方式：

```text
Command Log
Event Log
Snapshot
Sequence ID
```

恢复流程：

```text
1. Active Engine 故障
2. Standby Engine 检测心跳超时
3. Standby 读取最后 sequence_id
4. 从 WAL / Queue 补齐事件
5. Standby 切换为 Active
6. API / Adapter 路由切换
```

### 12.3 数据库 HA

PostgreSQL：

```text
主从复制
自动故障转移
每日全量备份
分钟级 WAL 归档
定期恢复演练
```

---

## 13. 低资源占用设计

为了满足低资源占用：

### 13.1 技术选择

```text
NATS 替代 Kafka
Go 静态编译
C++ 单二进制部署
PostgreSQL 单主多从
Redis 小规模缓存
ClickHouse 可选部署
Fabric 可选后置接入
```

### 13.2 服务合并策略

开发 / 小规模部署时，可以合并服务：

```text
aspira-api
  ├── user module
  ├── kyc module
  ├── risk module
  ├── payment module
  ├── fx module
  └── settlement module

aspira-engine-cpp

aspira-chain-service
```

这样 V2 最小部署只需要：

```text
Go Backend
C++ Engine
PostgreSQL
Redis
NATS
Blockchain Node / Hash Chain
Web Admin
```

### 13.3 最小资源配置

开发环境：

```text
CPU：4 Core
内存：8 GB
磁盘：100 GB SSD
```

准生产 Sandbox：

```text
CPU：8 Core
内存：16 GB
磁盘：300 GB SSD
```

小型生产验证：

```text
CPU：16 Core
内存：32 GB
磁盘：1 TB NVMe
```

---

## 14. 安全设计

### 14.1 API 安全

```text
HTTPS
JWT
mTLS
API Key
Request Signature
Timestamp
Nonce
Rate Limit
IP Allowlist
防重放攻击
```

签名示例：

```text
signature = HMAC_SHA256(
    method + path + timestamp + nonce + body_hash,
    api_secret
)
```

### 14.2 数据安全

```text
密码使用 Argon2id / bcrypt
KYC 信息加密存储
敏感字段脱敏
日志禁止输出证件号、银行卡号
对象存储加密
数据库备份加密
密钥使用 Vault / KMS
```

### 14.3 交易安全

```text
交易状态机校验
账户状态校验
幂等校验
金额正数校验
币种合法性校验
风控前置
大额人工审核
异常交易冻结
```

---

## 15. 可观测性设计

### 15.1 指标监控

Prometheus 指标：

```text
api_request_total
api_request_latency_ms
payment_created_total
payment_completed_total
payment_failed_total
risk_rejected_total
engine_command_latency_us
engine_tps
engine_error_total
settlement_lag_seconds
chain_submit_latency_ms
chain_submit_failed_total
```

### 15.2 日志

日志系统：

```text
Go：zap / zerolog
C++：spdlog
收集：Loki / Elasticsearch
```

日志必须包含：

```text
trace_id
request_id
payment_id
event_id
user_id_hash
service_name
latency
status
error_code
```

### 15.3 链路追踪

使用 OpenTelemetry：

```text
API Gateway
Payment Service
Risk Service
Engine Adapter
Settlement Service
Chain Service
```

---

## 16. 部署架构

### 16.1 Docker Compose V2 最小部署

```text
services:
  aspira-api
  aspira-engine
  aspira-chain-service
  postgres
  redis
  nats
  prometheus
  grafana
  web-admin
```

### 16.2 Kubernetes 部署

```text
namespace: aspira-pay

deployments:
  api-gateway
  payment-service
  user-service
  kyc-service
  risk-service
  fx-service
  settlement-service
  chain-service
  engine-adapter
  web-admin

statefulsets:
  postgres
  redis
  nats
  blockchain-node
  cpp-engine-active
  cpp-engine-standby

configmaps:
  service-config
  risk-rules
  currency-config

secrets:
  jwt-secret
  db-password
  api-signing-secret
  chain-private-key
```

---

## 17. 项目目录结构

```text
aspira-pay/
├── backend-go/
│   ├── cmd/
│   │   ├── api-gateway/
│   │   ├── user-service/
│   │   ├── kyc-service/
│   │   ├── risk-service/
│   │   ├── fx-service/
│   │   ├── payment-service/
│   │   ├── settlement-service/
│   │   └── chain-service/
│   ├── internal/
│   │   ├── domain/
│   │   │   ├── user/
│   │   │   ├── kyc/
│   │   │   ├── risk/
│   │   │   ├── payment/
│   │   │   ├── ledger/
│   │   │   └── settlement/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── transport/
│   │   ├── config/
│   │   ├── security/
│   │   └── observability/
│   ├── pkg/
│   │   ├── money/
│   │   ├── idgen/
│   │   ├── crypto/
│   │   ├── logger/
│   │   └── errors/
│   └── go.mod
│
├── engine-cpp/
│   ├── include/
│   │   ├── engine.hpp
│   │   ├── ledger.hpp
│   │   ├── command.hpp
│   │   ├── wal.hpp
│   │   └── publisher.hpp
│   ├── src/
│   │   ├── main.cpp
│   │   ├── engine.cpp
│   │   ├── ledger.cpp
│   │   ├── command_queue.cpp
│   │   ├── wal.cpp
│   │   └── publisher.cpp
│   ├── tests/
│   ├── proto/
│   └── CMakeLists.txt
│
├── proto/
│   ├── payment.proto
│   ├── engine.proto
│   ├── settlement.proto
│   └── chain.proto
│
├── web-admin/
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
│
├── deploy/
│   ├── docker-compose.yml
│   ├── k8s/
│   ├── helm/
│   └── scripts/
│
├── migrations/
│   ├── 001_init_users.sql
│   ├── 002_init_payments.sql
│   ├── 003_init_ledger.sql
│   └── 004_init_chain.sql
│
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── database.md
│   ├── engine.md
│   ├── blockchain.md
│   └── deployment.md
│
└── README.md
```

---

## 18. API 示例

### 18.1 创建支付订单

```http
POST /api/v2/payments
Content-Type: application/json
Authorization: Bearer <token>
Idempotency-Key: req_20260607_000001
```

```json
{
  "sender_user_id": "u_10001",
  "receiver_user_id": "u_20001",
  "source_currency": "USD",
  "target_currency": "JPY",
  "source_amount": 10000,
  "purpose": "family_support",
  "country_from": "US",
  "country_to": "JP"
}
```

响应：

```json
{
  "payment_id": "pay_20260607_000001",
  "status": "CREATED",
  "source_amount": 10000,
  "target_amount": 1560000,
  "fee_amount": 100,
  "fx_rate": "156.000000000000",
  "created_at": 1780848000
}
```

---

## 19. 性能目标

V2 Sandbox 目标：

| 指标 | 目标 |
|---|---:|
| API P95 延迟 | < 100 ms |
| 风控 P95 延迟 | < 50 ms |
| C++ 引擎内部处理延迟 | < 1 ms |
| 单机引擎 TPS | 10,000+ |
| 消息队列延迟 | < 10 ms |
| 本地账本落账延迟 | < 100 ms |
| 链上批量确认 | 秒级到分钟级 |
| 系统可用性 | 99.9% |
| 交易可追溯率 | 100% |
| 幂等覆盖率 | 100% |

---

## 20. 开发路线图

### 阶段 1：基础骨架

目标：跑通用户、账户、支付订单。

任务：

```text
1. 创建 Go 项目
2. 创建 PostgreSQL 表
3. 实现 User Service
4. 实现 Account Service
5. 实现 Payment Service
6. 实现基础 API Gateway
7. 实现 Web Admin 基础页面
```

### 阶段 2：KYC + 风控

目标：支付前置合规检查。

任务：

```text
1. KYC 状态机
2. 风险等级
3. 黑名单
4. 单笔限额
5. 日累计限额
6. 国家限制
7. 人工审核队列
```

### 阶段 3：C++ 引擎

目标：高性能处理交易命令。

任务：

```text
1. 定义 engine.proto
2. 实现 C++ command queue
3. 实现 in-memory ledger
4. 实现 freeze / execute / refund
5. 实现 WAL
6. 实现 engine.executed 事件发布
7. 压测 TPS 和延迟
```

### 阶段 4：清算账本

目标：复式记账和对账。

任务：

```text
1. 实现 ledger_entries
2. 实现 settlement_batches
3. 实现借贷平衡校验
4. 实现交易冲正
5. 实现对账报表
```

### 阶段 5：区块链审计

目标：交易过程链上可追溯。

任务：

```text
1. 实现 Hash Chain 版本
2. 实现 Merkle Tree
3. 实现 chain_events
4. 实现 batch root hash 上链
5. 接入 Hyperledger Fabric 或 Tendermint
6. 实现链上查询接口
```

### 阶段 6：高可用部署

目标：稳定运行。

任务：

```text
1. Docker Compose
2. Prometheus
3. Grafana
4. Loki
5. OpenTelemetry
6. C++ Engine Active / Standby
7. PostgreSQL 备份恢复
8. NATS JetStream 集群
```

---

## 21. V2 最小可运行版本

最小版本建议只保留：

```text
aspira-api-go
aspira-engine-cpp
aspira-chain-service
postgres
redis
nats
web-admin
```

最小交易链路：

```text
注册用户
  ↓
设置 KYC APPROVED
  ↓
充值模拟余额
  ↓
创建支付订单
  ↓
风控通过
  ↓
发送 engine.command
  ↓
C++ Engine 扣减 / 冻结
  ↓
Settlement 写复式账本
  ↓
Chain Service 写 Hash Chain
  ↓
订单完成
```

---

## 22. 关键技术原则

Aspira Pay V2 必须坚持：

```text
1. 金额不用 float / double，全部使用 int64 最小货币单位。
2. 交易接口必须幂等。
3. 账本只能追加，不能删除。
4. 所有交易必须有状态机。
5. 所有状态变化必须有事件。
6. 热路径不直接写慢系统。
7. C++ 引擎只负责高性能执行，不负责复杂业务。
8. Go 服务负责业务编排、合规、状态机和清算。
9. 区块链负责审计证明，不负责高频热路径。
10. 本地账本完成优先，链上确认最终一致。
11. 任何失败都必须可重试、可补偿、可追溯。
12. 所有关键事件都必须可重放。
13. 所有服务必须有健康检查。
14. 所有交易必须有 trace_id / request_id / payment_id / event_id。
15. 所有敏感数据必须加密或脱敏。
```

---

## 23. 推荐技术栈汇总

| 层级 | 技术 |
|---|---|
| API | Go + Gin / Fiber |
| RPC | gRPC |
| 消息队列 | NATS JetStream |
| C++ 引擎 | C++20 + CMake + spdlog + RocksDB |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 分析 | ClickHouse |
| 对象存储 | MinIO |
| 区块链 | Hyperledger Fabric / Tendermint / Hash Chain |
| 监控 | Prometheus + Grafana |
| 日志 | Loki / Elasticsearch |
| 链路追踪 | OpenTelemetry |
| 部署 | Docker Compose / Kubernetes |
| 前端 | React / Vue + TailwindCSS |

---

## 24. 总结

Aspira Pay V2 的核心架构是：

```text
Go 负责支付业务编排
C++ 负责高性能交易清算
PostgreSQL 负责业务账本
NATS 负责事件驱动
区块链负责交易审计和不可篡改证明
Redis 负责缓存和限流
Prometheus/Grafana 负责监控
Docker/Kubernetes 负责部署
```

最重要的设计思想：

> 支付系统的核心不是“转账”，而是 **合规、风控、幂等、账本、清算、审计、可恢复、可追溯**。