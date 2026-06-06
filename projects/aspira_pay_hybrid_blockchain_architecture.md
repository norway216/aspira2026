# Aspira Pay 跨境支付网关架构设计：支付阶段中心化，其余阶段去中心化区块链模式

**作者**：Aspira Studio  
**版本**：V2.0 Hybrid Blockchain Edition  
**定位**：B2B 金融基础设施 / 跨境支付 / 链上结算协作网络  
**核心原则**：  
> **支付清算与法币出入金阶段保持合规中心化；交易意向、报价、订单、状态流转、审计、凭证、对账、争议处理、数据溯源等非支付阶段尽量采用去中心化区块链交易模式。**

---

## 1. 设计目标

原系统是一个以 Go 网关、C++ 交易引擎、数据库、消息队列、审计系统为核心的中心化跨境支付网关。新版架构将系统改造为 **Hybrid Finance Architecture**：

- **支付阶段**：仍然通过银行、持牌支付机构、清算机构、SWIFT、本地支付网络等合规通道完成。
- **支付前阶段**：交易意向、报价、汇率锁定、手续费确认、合规预审、订单生成，使用链上合约和链下计算协同完成。
- **支付中阶段**：资金实际划转走中心化合规通道，但支付指令、状态证明、回执哈希写入链上。
- **支付后阶段**：对账、审计、凭证、争议处理、退款、追踪、备份、不可篡改日志使用区块链完成。
- **数据隐私**：敏感数据不上链，只上链哈希、承诺、状态、凭证索引和零知识证明结果。
- **高性能**：交易计算、风控评分、撮合、报价、汇率计算仍然由 C++/Go 高性能服务承担。
- **可监管**：KYC、AML、制裁名单、旅行规则、法币清算、税务报表和资金冻结接口仍由合规模块控制。

---

## 2. 核心设计边界

### 2.1 什么是“支付阶段”？

在本系统中，**支付阶段**指真实资金发生法币划转、银行清算、持牌支付机构扣款/入账、稳定币铸造/赎回、卡组织结算、本地支付网络结算的阶段。

这些环节不建议完全去中心化，因为它们涉及：

- 法币账户
- 银行存款
- 支付牌照
- KYC / AML
- 反恐融资
- 外汇合规
- 税务和报表
- 退款、拒付、冻结、司法协助

因此支付阶段采用 **中心化合规支付执行层**。

### 2.2 哪些阶段可以去中心化？

| 阶段 | 是否去中心化 | 说明 |
|---|---:|---|
| 用户身份注册 | 部分去中心化 | DID + 中心化 KYC 结果证明 |
| 商户准入 | 部分去中心化 | 商户资料中心化审核，审核结果上链 |
| 报价请求 | 去中心化 | 多做市商 / 多节点链上报价 |
| 汇率锁定 | 去中心化 | 报价承诺写入智能合约 |
| 交易意向 | 去中心化 | 订单意向链上登记 |
| 风控预审 | 混合 | 链下计算，结果哈希 / 证明上链 |
| 支付执行 | 中心化 | 银行、支付机构、清算网络执行 |
| 支付状态同步 | 去中心化 | 状态凭证、回执哈希上链 |
| 对账 | 去中心化 | 多方共识对账 |
| 审计 | 去中心化 | 不可篡改审计账本 |
| 争议处理 | 混合 | 仲裁流程链上，人工/机构裁决链下 |
| 退款 / 回滚 | 混合 | 退款资金走支付通道，状态走链上 |
| 报表归档 | 混合 | 报表链下存储，哈希上链 |

---

## 3. 总体架构

```text
┌─────────────────────────────────────────────────────────────────────┐
│                         Aspira Pay 前端层                            │
│  Web Dashboard / Merchant Portal / Admin Console / API Developer UI  │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                v
┌─────────────────────────────────────────────────────────────────────┐
│                         Go API Gateway                              │
│  TLS / mTLS / OAuth2 / JWT / Rate Limit / Request Signature / WAF    │
└─────────────────────────────────────────────────────────────────────┘
                                │
            ┌───────────────────┼───────────────────┐
            v                   v                   v
┌──────────────────┐  ┌──────────────────┐  ┌─────────────────────┐
│ C++ 交易计算引擎 │  │ Go 编排服务层     │  │ 合规与风控服务层     │
│ FX / Fee / Route │  │ Saga / Workflow   │  │ KYC / AML / Sanction │
└──────────────────┘  └──────────────────┘  └─────────────────────┘
            │                   │                   │
            └───────────────────┼───────────────────┘
                                v
┌─────────────────────────────────────────────────────────────────────┐
│                    区块链交易协作层                                  │
│ Smart Contracts / Order Registry / Quote Commit / State Machine      │
│ Audit Ledger / Dispute Contract / Settlement Proof / DID Registry    │
└─────────────────────────────────────────────────────────────────────┘
                                │
            ┌───────────────────┼───────────────────┐
            v                   v                   v
┌──────────────────┐  ┌──────────────────┐  ┌─────────────────────┐
│ 链下数据存储层   │  │ 消息队列事件层   │  │ 中心化支付执行层     │
│ PostgreSQL/MinIO │  │ Kafka/NATS       │  │ Bank/PSP/SWIFT/Card  │
└──────────────────┘  └──────────────────┘  └─────────────────────┘
                                │
                                v
┌─────────────────────────────────────────────────────────────────────┐
│                   审计、监控、报表、风控回放层                        │
│ ELK / Prometheus / Grafana / SIEM / Chain Explorer / Reconciliation │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. 技术选型

### 4.1 后端语言

| 模块 | 推荐语言 | 原因 |
|---|---|---|
| API 网关 | Go | 高并发网络服务、开发效率高 |
| 工作流编排 | Go | 易于实现 Saga、状态机、分布式任务 |
| 交易计算引擎 | C++ | 低延迟、高吞吐、适合汇率/手续费/撮合计算 |
| 链上交互服务 | Go / Rust | Go 适合集成服务，Rust 适合高安全节点组件 |
| 智能合约 | Solidity / Move / Rust | 取决于链选择 |
| 风控模型服务 | Go / C++ / Python | 实时部分用 C++/Go，离线模型可用 Python |
| 前端面板 | React / Vue | 管理后台和商户平台 |
| 数据分析 | Python / SQL | 报表、离线风控、交易分析 |

### 4.2 区块链类型选择

建议不要直接使用完全公链作为核心业务账本，而采用：

> **联盟链 / 许可链为主，公链锚定为辅。**

| 类型 | 用途 | 优点 | 风险 |
|---|---|---|---|
| 联盟链 / 许可链 | 订单、状态、审计、对账 | 可控、隐私好、性能高 | 去中心化程度较弱 |
| 公链 L2 | 跨机构凭证锚定 | 公开可验证 | 成本、合规、隐私风险 |
| 私有链 | 内部开发测试 | 快速、低成本 | 公信力弱 |
| 混合链 | 推荐方案 | 性能、合规、可信度平衡 | 架构复杂 |

推荐链架构：

```text
业务联盟链：Aspira Consortium Chain
    ├── 参与方：Aspira Pay、合作银行、支付机构、做市商、审计节点
    ├── 共识：IBFT / Tendermint / HotStuff 类 BFT 共识
    ├── 用途：订单、报价、状态、审计、对账
    └── 隐私：通道 / 子账本 / 加密字段 / 零知识证明

公链锚定层：
    ├── 周期性 Merkle Root 上链
    ├── 用途：防止联盟链内部篡改
    └── 不存储任何敏感交易明细
```

---

## 5. 模块设计

## 5.1 Go API 网关层

### 功能

- 接收商户 API 请求
- TLS / mTLS 加密通信
- 请求签名验证
- 幂等控制
- 限流
- 黑名单拦截
- JWT / OAuth2 认证
- 商户权限校验
- 请求路由到链上交易服务或支付执行服务

### 核心接口

```http
POST /api/v1/quote
POST /api/v1/order/create
POST /api/v1/order/commit
POST /api/v1/payment/execute
GET  /api/v1/payment/status/{payment_id}
POST /api/v1/refund
GET  /api/v1/audit/proof/{tx_id}
GET  /api/v1/reconciliation/report
```

### 幂等设计

每笔请求必须包含：

```json
{
  "merchant_id": "MCH_10001",
  "request_id": "uuid",
  "nonce": "random",
  "timestamp": 1760000000,
  "signature": "HMAC-SHA256 or Ed25519"
}
```

网关根据 `merchant_id + request_id` 建立幂等表，防止重复扣款。

---

## 5.2 C++ 高性能交易计算引擎

### 定位

C++ 引擎不直接完成法币支付，而负责支付前后的高性能计算：

- 汇率计算
- 手续费计算
- 路由选择
- 做市商报价排序
- 交易限额计算
- 流动性预估
- 风险评分预计算
- 交易状态校验
- 链下 Merkle Tree 构建
- 批量对账计算

### 内部结构

```text
C++ Trading Compute Engine
├── Request Parser
├── Lock-Free Queue
├── FX Rate Engine
├── Fee Engine
├── Liquidity Router
├── Risk Pre-Score Engine
├── Quote Aggregator
├── Merkle Batch Builder
├── State Validator
└── Result Publisher
```

### 性能设计

- 每个交易请求进入 lock-free ring buffer
- 读多写少数据使用 RCU / Copy-on-write
- 汇率表使用内存快照
- 手续费规则预编译成决策树
- 风控规则分为同步硬规则和异步软规则
- 批量计算后写入 Kafka / NATS
- 单笔交易只做必要同步计算，复杂分析异步处理

---

## 5.3 区块链交易协作层

这是新版架构的核心。

### 5.3.1 链上合约模块

```text
Smart Contract Suite
├── DIDRegistry              身份与机构注册
├── MerchantRegistry         商户准入状态
├── QuoteCommitment          报价承诺
├── OrderRegistry            订单登记
├── PaymentStateMachine      支付状态机
├── SettlementProofRegistry  支付回执证明
├── AuditLedger              审计日志
├── DisputeResolution        争议处理
├── RefundRegistry           退款状态
└── MerkleAnchor             批量数据锚定
```

### 5.3.2 订单状态机

```text
CREATED
  ↓
QUOTE_LOCKED
  ↓
COMPLIANCE_PRECHECKED
  ↓
PAYMENT_PENDING
  ↓
PAYMENT_EXECUTING
  ↓
PAYMENT_CONFIRMED
  ↓
SETTLEMENT_PROOFED
  ↓
RECONCILED
  ↓
CLOSED
```

异常状态：

```text
RISK_REJECTED
PAYMENT_FAILED
REFUND_PENDING
REFUNDED
DISPUTED
FROZEN
MANUAL_REVIEW
CANCELLED
```

### 5.3.3 链上存储内容

链上只存：

| 数据 | 是否上链 | 说明 |
|---|---:|---|
| 订单 ID | 是 | 唯一标识 |
| 商户 ID 哈希 | 是 | 不暴露真实商户信息 |
| 用户 ID 哈希 | 是 | 不暴露用户隐私 |
| 交易金额哈希 | 是 | 金额明文不上链 |
| 币种 | 可选 | 可按合规要求脱敏 |
| 报价承诺 | 是 | 防止报价方反悔 |
| 支付状态 | 是 | 状态机流转 |
| 支付回执哈希 | 是 | 回执文件链下存储 |
| 风控结果哈希 | 是 | 不暴露规则 |
| KYC 结果证明 | 是 | 只上证明不上明文 |
| 审计日志哈希 | 是 | 保证不可篡改 |
| 完整用户资料 | 否 | 存加密数据库 |
| 银行账号 | 否 | 存加密数据库 / HSM |
| 身份证件 | 否 | 存加密对象存储 |
| 交易明细全文 | 否 | 存链下数据库 |

---

## 5.4 中心化支付执行层

### 作用

真实资金划转仍然由中心化支付通道执行。

```text
Payment Execution Layer
├── Bank Connector
├── PSP Connector
├── SWIFT Connector
├── Card Network Connector
├── Local Payment Rail Connector
├── Stablecoin Issuer Connector
├── FX Provider Connector
└── Settlement Callback Handler
```

### 支付执行流程

```text
链上订单已创建
  ↓
链上报价已锁定
  ↓
风控与合规预审通过
  ↓
Go Orchestrator 生成支付指令
  ↓
支付执行层调用银行 / PSP / 清算网络
  ↓
收到支付回执
  ↓
回执原文加密存储到对象存储
  ↓
回执哈希写入链上 SettlementProofRegistry
  ↓
PaymentStateMachine 状态更新为 PAYMENT_CONFIRMED
```

### 支付阶段为何不去中心化

- 法币支付需要银行账户体系
- 跨境支付涉及外汇与牌照
- 资金清算需要合规机构
- 退款、拒付、冻结需要责任主体
- 监管报送需要可识别主体
- 消费者保护需要明确责任边界

---

## 5.5 合规与风控模块

### 合规模块

```text
Compliance Service
├── KYC Verification
├── KYB Verification
├── AML Screening
├── Sanctions Screening
├── PEP Screening
├── Travel Rule Data Exchange
├── Transaction Limit Control
├── Country / Region Policy Engine
├── Suspicious Activity Report
└── Manual Review Console
```

### 风控模块

```text
Risk Engine
├── Rule Engine
├── Velocity Check
├── Device Fingerprint
├── IP / Geo Risk
├── Amount Pattern Detection
├── Merchant Risk Profile
├── Graph Risk Detection
├── AI Fraud Scoring
└── Case Management
```

### 链上表达方式

合规和风控的具体规则不上链，只上链：

```json
{
  "order_id": "ORD_xxx",
  "risk_result_hash": "0xabc...",
  "compliance_result_hash": "0xdef...",
  "risk_level": "LOW / MEDIUM / HIGH",
  "review_required": false,
  "proof_timestamp": 1760000000
}
```

高级版本可以使用零知识证明表达：

> 系统证明“该用户通过 KYC、未命中制裁名单、未超过限额”，但不公开用户真实身份和完整交易数据。

---

## 5.6 链下数据层

### 数据库

```text
PostgreSQL
├── merchant_profile
├── customer_profile
├── order_private_data
├── payment_instruction
├── bank_receipt
├── compliance_case
├── risk_case
├── refund_record
├── reconciliation_record
└── audit_private_payload
```

### 对象存储

```text
MinIO / S3
├── encrypted_kyc_files
├── encrypted_payment_receipts
├── encrypted_bank_statements
├── encrypted_reconciliation_reports
├── encrypted_audit_reports
└── encrypted_dispute_documents
```

### 加密策略

- 敏感字段使用 AES-256-GCM 加密
- 密钥托管在 HSM / KMS
- 每个商户 / 每个用户 / 每类数据使用不同数据密钥
- 对象存储文件先加密再上传
- 数据库只保存密文和密钥索引
- 链上只保存哈希，不保存明文

---

## 5.7 消息队列与事件总线

### Kafka / NATS Topic 设计

```text
quote.requested
quote.generated
quote.committed.onchain
order.created
order.state.updated
risk.prechecked
compliance.prechecked
payment.instruction.created
payment.executed
payment.callback.received
settlement.proof.created
reconciliation.started
reconciliation.completed
refund.requested
refund.completed
audit.event.created
dispute.opened
dispute.closed
```

### 事件原则

- 每个事件有全局唯一 `event_id`
- 每个事件绑定 `order_id`
- 每个事件有 `prev_event_hash`
- 每个事件可生成 Merkle Root
- 批量事件根哈希上链
- 所有消费者必须幂等消费

---

## 6. 完整交易流程

## 6.1 支付前：去中心化交易协作流程

```text
1. 商户调用 /quote 请求报价
2. Go 网关验签、限流、鉴权
3. C++ 引擎计算候选汇率、手续费、路由
4. 多报价节点 / 做市商提交 Quote Commitment
5. QuoteCommitment 合约锁定报价
6. 商户确认报价
7. OrderRegistry 创建链上订单
8. 风控与合规模块完成预审
9. 合规结果哈希写入链上
10. PaymentStateMachine 更新为 COMPLIANCE_PRECHECKED
```

### 结果

此时交易意向、报价承诺、订单状态已经去中心化记录，但资金还没有真正划转。

---

## 6.2 支付中：中心化支付执行流程

```text
1. Orchestrator 检查链上状态是否满足支付条件
2. 生成支付指令
3. 调用银行 / PSP / SWIFT / 本地支付网络
4. 支付执行层等待回调或主动轮询
5. 支付成功后保存支付回执
6. 计算回执哈希
7. SettlementProofRegistry 写入支付证明
8. PaymentStateMachine 更新为 PAYMENT_CONFIRMED
```

### 结果

真实资金划转由合规机构完成，但支付状态和回执证明由链上记录。

---

## 6.3 支付后：去中心化对账与审计流程

```text
1. 银行流水进入 Reconciliation Service
2. 交易系统流水进入 Reconciliation Service
3. 链上 PaymentStateMachine 状态进入 Reconciliation Service
4. 三方数据进行匹配：
   - 内部订单
   - 银行/PSP 回执
   - 链上状态
5. 生成对账报告
6. 报告加密存储
7. 报告哈希写入 AuditLedger
8. 异常订单进入争议合约 DisputeResolution
```

### 结果

对账与审计具备不可篡改的链上证明。

---

## 7. 智能合约设计

## 7.1 OrderRegistry

### 功能

- 创建订单
- 绑定商户
- 绑定报价承诺
- 记录订单状态
- 防止重复订单

### 伪代码

```solidity
contract OrderRegistry {
    struct Order {
        bytes32 orderId;
        bytes32 merchantHash;
        bytes32 customerHash;
        bytes32 amountCommitment;
        bytes32 quoteId;
        uint256 createdAt;
        OrderStatus status;
    }

    mapping(bytes32 => Order) public orders;

    function createOrder(
        bytes32 orderId,
        bytes32 merchantHash,
        bytes32 customerHash,
        bytes32 amountCommitment,
        bytes32 quoteId
    ) external onlyAuthorizedNode {
        require(orders[orderId].createdAt == 0, "ORDER_EXISTS");
        orders[orderId] = Order(
            orderId,
            merchantHash,
            customerHash,
            amountCommitment,
            quoteId,
            block.timestamp,
            OrderStatus.CREATED
        );
    }
}
```

---

## 7.2 QuoteCommitment

### 功能

- 锁定报价
- 防止报价后反悔
- 记录报价有效期
- 支持多个报价节点竞争

### 核心字段

```text
quote_id
provider_id_hash
source_currency_hash
target_currency_hash
rate_commitment
fee_commitment
expire_at
signature
```

---

## 7.3 PaymentStateMachine

### 功能

- 约束状态流转
- 防止非法跳转
- 记录状态更新时间
- 提供外部查询接口

### 状态转移规则

| 当前状态 | 允许下一状态 |
|---|---|
| CREATED | QUOTE_LOCKED / CANCELLED |
| QUOTE_LOCKED | COMPLIANCE_PRECHECKED / RISK_REJECTED |
| COMPLIANCE_PRECHECKED | PAYMENT_PENDING / MANUAL_REVIEW |
| PAYMENT_PENDING | PAYMENT_EXECUTING / CANCELLED |
| PAYMENT_EXECUTING | PAYMENT_CONFIRMED / PAYMENT_FAILED |
| PAYMENT_CONFIRMED | SETTLEMENT_PROOFED |
| SETTLEMENT_PROOFED | RECONCILED / DISPUTED |
| RECONCILED | CLOSED |

---

## 7.4 SettlementProofRegistry

### 功能

- 存储支付回执哈希
- 绑定链下回执文件索引
- 标记支付通道
- 用于审计和对账

### 字段

```text
order_id
payment_channel
receipt_hash
receipt_uri_hash
confirmed_at
executor_node
signature
```

---

## 7.5 AuditLedger

### 功能

- 存储审计事件哈希
- 存储批量 Merkle Root
- 支持审计证明验证
- 支持跨链锚定

```text
audit_event_id
order_id
event_type
event_hash
prev_event_hash
merkle_root
timestamp
operator_hash
```

---

## 8. 数据一致性设计

系统采用 **链上状态机 + 链下事务 + Saga 补偿机制**。

### 8.1 为什么不使用单纯 2PC？

跨境支付涉及银行、PSP、区块链、数据库、消息队列、对象存储，不可能全部纳入一个强一致事务。

因此采用：

```text
本地事务 + 事件驱动 + 状态机约束 + 幂等消费 + 补偿流程
```

### 8.2 Saga 流程

```text
CreateOrder
  ↓
LockQuote
  ↓
ComplianceCheck
  ↓
CreatePaymentInstruction
  ↓
ExecutePayment
  ↓
WriteSettlementProof
  ↓
Reconcile
  ↓
CloseOrder
```

### 8.3 补偿动作

| 失败位置 | 补偿动作 |
|---|---|
| 报价锁定失败 | 订单取消 |
| 合规预审失败 | 状态改为 RISK_REJECTED |
| 支付指令生成失败 | 重新生成 / 人工处理 |
| 支付执行失败 | 状态改为 PAYMENT_FAILED |
| 支付成功但链上写入失败 | Outbox 重试写链 |
| 链上成功但数据库失败 | 从链上事件回放恢复 |
| 对账失败 | 进入 DISPUTED |
| 退款失败 | 保持 REFUND_PENDING 并告警 |

---

## 9. 安全架构

## 9.1 通信安全

- 外部 API：HTTPS / TLS 1.3
- 内部服务：mTLS
- 节点通信：mTLS + 节点证书
- 商户请求：HMAC-SHA256 / Ed25519 签名
- 回调通知：签名 + 时间戳 + nonce

## 9.2 密钥管理

```text
HSM / KMS
├── Merchant API Key
├── Service Signing Key
├── Blockchain Node Key
├── Data Encryption Key
├── Receipt Encryption Key
├── JWT Signing Key
└── Backup Recovery Key
```

原则：

- 私钥不落盘
- 高权限操作需要多签
- 链上合约升级需要多签
- 生产环境禁用万能管理员
- 密钥定期轮换
- 关键操作进入审计

## 9.3 链上安全

- 合约必须经过审计
- 合约升级使用 Timelock + Multisig
- 节点准入使用证书
- 关键状态转移需要授权节点签名
- 金额不上链明文
- 隐私数据不上链
- 所有链下数据有哈希校验

---

## 10. 隐私保护设计

### 10.1 敏感数据不上链

链上不存储：

- 姓名
- 身份证件
- 护照
- 银行卡号
- 银行账号
- 详细地址
- 手机号
- 邮箱
- 完整交易备注
- 完整银行回执

### 10.2 哈希与承诺

金额、身份、商户、订单详情使用：

```text
commitment = Hash(value + salt)
```

只有持有原文和 salt 的授权方可以证明该承诺对应某笔交易。

### 10.3 零知识证明方向

后续可以引入 ZKP：

- 证明用户通过 KYC，但不暴露身份
- 证明交易金额低于限额，但不暴露金额
- 证明不在黑名单，但不暴露筛查细节
- 证明订单被正确对账，但不公开银行流水

---

## 11. 节点角色设计

```text
Aspira Consortium Chain Nodes
├── Aspira Core Node
├── Bank Partner Node
├── PSP Partner Node
├── FX Provider Node
├── Liquidity Provider Node
├── Auditor Node
├── Regulator Observer Node
└── Disaster Recovery Node
```

### 节点权限

| 节点 | 权限 |
|---|---|
| Aspira Core Node | 创建订单、更新状态、发起支付证明 |
| Bank Partner Node | 确认支付回执、参与对账 |
| PSP Partner Node | 确认支付回调、参与对账 |
| FX Provider Node | 提交报价承诺 |
| Liquidity Provider Node | 提供流动性状态 |
| Auditor Node | 读取审计账本、验证哈希 |
| Regulator Observer Node | 只读监管观察 |
| DR Node | 灾备同步 |

---

## 12. 交易 ID 与哈希链设计

### 12.1 全局 ID

```text
order_id      = ORD_{region}_{timestamp}_{snowflake}
payment_id    = PAY_{channel}_{timestamp}_{snowflake}
quote_id      = QTE_{provider}_{timestamp}_{snowflake}
audit_id      = AUD_{timestamp}_{snowflake}
event_id      = EVT_{timestamp}_{snowflake}
```

### 12.2 事件哈希链

每个事件包含上一个事件哈希：

```json
{
  "event_id": "EVT_001",
  "order_id": "ORD_001",
  "event_type": "PAYMENT_CONFIRMED",
  "payload_hash": "0xabc",
  "prev_event_hash": "0xdef",
  "timestamp": 1760000000,
  "signature": "0x..."
}
```

这样可以形成链下事件链，并将 Merkle Root 批量写入链上。

---

## 13. 对账系统设计

### 13.1 三账对账

```text
内部订单账
    +
支付通道账
    +
链上状态账
    =
最终对账结果
```

### 13.2 对账维度

- order_id
- payment_id
- amount
- currency
- channel
- status
- receipt_hash
- settlement_date
- merchant_id
- fee
- FX rate

### 13.3 异常类型

| 异常 | 处理 |
|---|---|
| 内部成功，通道失败 | 退款 / 冲正 |
| 通道成功，链上未确认 | 重试写链 |
| 链上确认，数据库缺失 | 事件回放恢复 |
| 金额不一致 | 冻结进入人工审核 |
| 手续费不一致 | 调整账务分录 |
| 汇率不一致 | 按报价承诺仲裁 |
| 重复回调 | 幂等忽略 |
| 超时无回执 | 轮询 + 告警 |

---

## 14. 退款与回滚设计

### 14.1 重要原则

区块链交易本身不可删除，因此系统中的“回滚”不是抹掉历史，而是通过反向事件修正状态。

```text
错误交易记录仍然存在
    ↓
生成 REFUND / REVERSAL 事件
    ↓
退款资金通过支付执行层原路或替代路径退回
    ↓
退款证明写入链上
    ↓
最终状态变为 REFUNDED / REVERSED
```

### 14.2 退款流程

```text
退款申请
  ↓
RefundRegistry 创建退款记录
  ↓
风控检查
  ↓
支付执行层执行退款
  ↓
保存退款回执
  ↓
退款回执哈希上链
  ↓
订单状态更新为 REFUNDED
```

---

## 15. 管理后台设计

### 15.1 页面模块

```text
Aspira Pay Admin Console
├── Overview Dashboard
├── Live Transactions
├── Blockchain Explorer
├── Payment Channel Monitor
├── Risk & Compliance Cases
├── Reconciliation Center
├── Audit Proof Center
├── Dispute Center
├── Merchant Management
├── Node Management
├── Key Management
└── System Observability
```

### 15.2 视觉风格

B2B 金融基础设施风格建议：

- 主色：深墨蓝 / 深海军蓝
- 辅色：科技蓝 / 冰蓝
- 强调色：少量紫蓝渐变
- 字体：几何感、线条清晰、间距克制
- 风格：稳重、专业、可信、机构级
- 避免：过度粉色、过度糖果色、过强娱乐化视觉

---

## 16. 部署架构

```text
Kubernetes Cluster
├── ingress-nginx / Envoy Gateway
├── go-api-gateway
├── go-orchestrator
├── cpp-trading-engine
├── compliance-service
├── risk-engine
├── payment-executor
├── blockchain-indexer
├── blockchain-writer
├── reconciliation-service
├── audit-service
├── admin-console
├── kafka-cluster
├── postgresql-cluster
├── redis-cluster
├── minio-cluster
├── prometheus
├── grafana
├── elasticsearch
└── blockchain-nodes
```

### 高可用策略

- API Gateway 多副本
- C++ 引擎按交易类型分片
- Kafka 三节点以上
- PostgreSQL 主从 + 自动故障切换
- 区块链节点至少 4 个 BFT 节点
- 跨可用区部署
- 对象存储多副本
- 灾备节点异地部署

---

## 17. 灾备与恢复

### 17.1 数据恢复来源

系统可以从以下来源恢复：

1. PostgreSQL WAL
2. Kafka 事件日志
3. 区块链状态账本
4. 对象存储加密文件
5. 审计 Merkle Root
6. 支付通道原始回执

### 17.2 恢复策略

```text
数据库损坏：
    从 PostgreSQL 备份 + 链上状态 + Kafka 事件恢复

链节点异常：
    从其他联盟链节点同步

对象存储异常：
    从多副本对象存储恢复

支付通道回调丢失：
    主动轮询通道接口 + 对账补偿

Kafka 消息丢失：
    从数据库 Outbox 表重新投递
```

---

## 18. 合规边界

本架构虽然引入去中心化区块链交易模式，但不应该绕开合规支付体系。

必须保留：

- KYC / KYB
- AML / CFT
- 制裁名单筛查
- 可疑交易监测
- 交易限额
- 税务报表
- 监管报送
- 资金冻结接口
- 司法协助接口
- 客诉与退款机制
- 数据隐私保护
- 持牌支付通道

系统的正确定位是：

> **使用区块链提升跨机构协作、审计可信度、对账效率和交易透明度，而不是用区块链逃避监管。**

---

## 19. 与原始中心化架构的变化对比

| 模块 | 原架构 | 新架构 |
|---|---|---|
| 订单生成 | 数据库写入 | 链上订单登记 + 链下详情 |
| 报价 | 中心化计算 | 多节点报价承诺 |
| 支付执行 | 中心化 | 仍然中心化 |
| 支付状态 | 数据库状态 | 链上状态机 + 数据库缓存 |
| 审计 | ELK 日志 | ELK + 链上哈希证明 |
| 对账 | 数据库对账 | 内部账 + 通道账 + 链上账 |
| 回滚 | 数据库事务回滚 | 反向事件 + 退款状态 |
| 数据存储 | PostgreSQL 为主 | PostgreSQL + 链上证明 + 对象存储 |
| 隐私 | 数据库加密 | 数据库加密 + 链上哈希承诺 |
| 多机构协作 | API 集成 | 联盟链节点协作 |

---

## 20. 推荐开发路线

### 第一阶段：中心化系统增强

- Go API Gateway
- C++ 交易计算引擎
- PostgreSQL
- Kafka
- 支付执行层
- 基础风控与合规
- 管理后台

### 第二阶段：链上审计

- AuditLedger
- Merkle Root 上链
- 支付回执哈希上链
- 链上浏览器
- 审计证明查询

### 第三阶段：链上订单与报价

- OrderRegistry
- QuoteCommitment
- PaymentStateMachine
- 做市商报价节点
- 商户订单链上查询

### 第四阶段：联盟链多机构协作

- 银行节点
- PSP 节点
- 审计节点
- 监管观察节点
- 多方对账

### 第五阶段：隐私增强

- DID
- Verifiable Credential
- 零知识 KYC 证明
- 金额承诺
- 跨链锚定

---

## 21. 最终架构总结

新版 Aspira Pay 的核心不是“完全去中心化支付”，而是：

> **支付执行合规中心化，交易协作、订单状态、报价承诺、审计证明、对账凭证去中心化。**

这种设计更适合真实 B2B 金融基础设施，因为它同时保留：

- 金融合规
- 支付可执行性
- 多机构协作
- 审计可信度
- 数据可追溯
- 系统高性能
- 隐私保护
- 未来扩展到 Web3 / 稳定币 / 代币化结算的能力

最终系统可以被理解为：

```text
Aspira Pay =
    高性能支付网关
  + 中心化合规支付执行层
  + 去中心化交易协作网络
  + 链上审计与对账账本
  + 隐私保护数据架构
  + B2B 金融基础设施控制台
```
