# RK3568 Hardware Wallet 安全增强架构设计方案

> 版本：v1.1  
> 目标平台：RK3568 / 嵌入式 Linux / Ubuntu / Debian  
> 技术栈：C++20 + Qt/QML + SQLite + OpenSSL + spdlog  
> 核心增强：密码连续输入错误 3 次后，安全删除当前用户对应的钱包数据。

---

## 1. 系统目标

本系统是在现有 **RK3568 Hardware Wallet** 架构基础上扩展而来，目标是设计一个可以运行在 RK3568 或普通 Ubuntu/Debian 环境中的简单硬件钱包系统。

系统使用：

- C++20 实现核心业务逻辑；
- QML / Qt Quick 实现嵌入式 UI；
- SQLite 存储用户、钱包、交易和审计日志；
- OpenSSL 实现密码派生、钱包加密、交易签名与完整性校验；
- spdlog 实现运行日志；
- Repository + Service + ViewModel + QML 的分层架构。

核心功能包括：

1. 用户注册、登录、登出；
2. 用户密码派生 KEK；
3. KEK 解密 DEK；
4. DEK 解密钱包种子；
5. 钱包种子派生 Ed25519 密钥对；
6. 用户之间模拟转账；
7. 交易签名与验签；
8. 审计日志哈希链；
9. 加密备份与恢复；
10. 密码连续输入错误 3 次后，销毁当前用户对应的钱包数据。

---

## 2. 安全策略总览

### 2.1 错误 3 次后删除什么？

不建议只执行：

```sql
DELETE FROM wallets WHERE user_id = ?;
```

因为 SQLite 的 WAL、journal、磁盘页缓存中可能仍然残留旧数据。

更合理的策略是：

```text
密码错误 3 次
  ↓
1. 销毁当前用户的 encryptedDEK
2. 删除当前用户的钱包 seed / wallet / transaction 数据
3. 清理当前会话内存中的 DEK
4. 写入审计日志
5. 执行 SQLite checkpoint / secure_delete / VACUUM
```

真正关键的是：

```text
只要 DEK 被破坏，即使数据库中残留 encryptedSeed，也无法恢复钱包私钥。
```

所以本系统采用：

```text
加密擦除 + 业务删除 + 审计记录 + SQLite 清理
```

---

## 3. 密钥层级设计

```text
用户密码
  ↓ PBKDF2-HMAC-SHA256
KEK，Key Encryption Key
  ↓ AES-256-GCM 解密
DEK，Data Encryption Key
  ↓ AES-256-GCM 解密
钱包种子 seed
  ↓ Ed25519 派生
公钥 / 私钥
  ↓
交易签名 / 验签
```

详细结构：

```text
                           用户密码 User Password
                                   │
                        PBKDF2-HMAC-SHA256
                        + random salt
                                   │
                            KEK，32 bytes
                                   │
                  ┌────────────────┴────────────────┐
                  │                                 │
           AES-256-GCM                         AES-256-GCM
           wrap DEK                            unwrap DEK
           注册时                              登录时
                  │                                 │
                  v                                 v
          encryptedDEK                         plaintext DEK
          存储于 DB                             仅存在内存中
                                                    │
                                            AES-256-GCM
                                                    │
                                            encryptedSeed
                                                    │
                                            wallet seed
                                                    │
                                            Ed25519 KeyPair
```

安全原则：

1. 用户密码不直接加密钱包种子；
2. 用户密码只用于派生 KEK；
3. KEK 只用于加密 / 解密 DEK；
4. DEK 用于加密 / 解密钱包种子；
5. DEK 只存在于当前登录会话内存中；
6. 错误 3 次时优先销毁 encryptedDEK；
7. 即使 encryptedSeed 残留，DEK 被销毁后也无法恢复钱包私钥。

---

## 4. 安全参数设计

建议在 `src/app/Constants.h` 中定义：

```cpp
namespace AppConstants {
    constexpr int MIN_PASSWORD_LENGTH = 8;

    // 登录安全策略
    constexpr int MAX_FAILED_ATTEMPTS = 3;

    // 是否启用自毁策略
    constexpr bool WIPE_WALLET_ON_MAX_FAILED_ATTEMPTS = true;

    // 是否在钱包删除时连用户一起删除
    constexpr bool DELETE_USER_ON_WIPE = false;

    // SQLite 安全清理策略
    constexpr bool ENABLE_SQLITE_SECURE_DELETE = true;
    constexpr bool ENABLE_VACUUM_AFTER_WIPE = true;

    // 会话超时策略
    constexpr int DEFAULT_AUTO_LOCK_SECONDS = 300;
}
```

默认策略：

```text
密码错误 3 次：删除钱包数据，但保留用户账户。
```

用户再次登录时提示：

```text
当前用户的钱包数据已被安全删除，请通过加密备份恢复。
```

如果需要更激进的安全策略，可以设置：

```cpp
DELETE_USER_ON_WIPE = true;
```

这表示错误 3 次后，连用户账户也一起删除。

---

## 5. 数据库设计增强

### 5.1 users 表

建议增加以下字段：

```sql
wallet_wiped INTEGER NOT NULL DEFAULT 0,
wiped_at INTEGER NOT NULL DEFAULT 0,
backup_created INTEGER NOT NULL DEFAULT 0,
last_backup_at INTEGER NOT NULL DEFAULT 0
```

完整 users 表设计：

```sql
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,

    encrypted_dek BLOB NOT NULL,
    password_verifier BLOB NOT NULL,
    salt BLOB NOT NULL,

    kdf_ops_limit INTEGER NOT NULL,
    kdf_mem_limit INTEGER NOT NULL,

    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until INTEGER NOT NULL DEFAULT 0,

    wallet_wiped INTEGER NOT NULL DEFAULT 0,
    wiped_at INTEGER NOT NULL DEFAULT 0,

    backup_created INTEGER NOT NULL DEFAULT 0,
    last_backup_at INTEGER NOT NULL DEFAULT 0,

    created_at INTEGER NOT NULL,
    last_login_at INTEGER NOT NULL DEFAULT 0
);
```

字段说明：

| 字段 | 含义 |
|---|---|
| `encrypted_dek` | 使用 KEK 加密后的 DEK |
| `password_verifier` | 密码验证哈希 |
| `salt` | 密码盐值 |
| `failed_attempts` | 连续密码错误次数 |
| `wallet_wiped` | 钱包是否已被安全删除 |
| `wiped_at` | 钱包删除时间 |
| `backup_created` | 是否创建过备份 |
| `last_backup_at` | 最近一次备份时间 |

---

### 5.2 wallets 表

```sql
CREATE TABLE IF NOT EXISTS wallets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,

    encrypted_seed BLOB NOT NULL,
    public_key BLOB NOT NULL UNIQUE,

    balance REAL NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

说明：

- 每个用户只有一个钱包；
- `encrypted_seed` 使用 DEK 加密；
- `public_key` 是 Ed25519 公钥；
- 删除用户时可级联删除钱包。

---

### 5.3 transactions 表

```sql
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    wallet_id TEXT NOT NULL,
    to_public_key TEXT NOT NULL,
    amount REAL NOT NULL,
    fee REAL NOT NULL,
    signature BLOB NOT NULL,
    nonce TEXT NOT NULL UNIQUE,
    status INTEGER NOT NULL,
    memo TEXT,
    created_at INTEGER NOT NULL,

    FOREIGN KEY(wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
);
```

策略：

```text
钱包被删除时，当前用户相关交易历史也删除。
```

如果项目希望保留交易审计记录，可以将审计日志独立保留，不删除 `audit_logs`。

---

### 5.4 audit_logs 表

```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action TEXT NOT NULL,
    user_id TEXT,
    details TEXT,
    previous_hash TEXT NOT NULL,
    hash TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
```

新增审计动作：

```cpp
static const char* ACTION_LOGIN_FAILED = "LOGIN_FAILED";
static const char* ACTION_WALLET_WIPED = "WALLET_WIPED";
static const char* ACTION_SECURITY_SELF_DESTRUCT = "SECURITY_SELF_DESTRUCT";
```

审计日志建议不要删除，因为它用于证明发生过安全销毁事件。

---

## 6. Repository 层增强设计

### 6.1 UserRepository 新增方法

```cpp
class UserRepository {
public:
    Result<void> incrementFailedAttempts(const QString& userId);
    Result<void> resetFailedAttempts(const QString& userId);

    Result<void> markWalletWiped(const QString& userId, qint64 wipedAt);

    Result<void> destroyEncryptedDEK(const QString& userId);

    Result<void> markBackupCreated(const QString& userId, qint64 backupAt);

    Result<void> remove(const QString& userId);
};
```

### 6.2 destroyEncryptedDEK 设计

不要简单设置为空，因为字段可能是 `NOT NULL`。建议写入随机垃圾数据：

```cpp
Result<void> UserRepository::destroyEncryptedDEK(const QString& userId)
{
    QByteArray randomGarbage(64, 0);
    RAND_bytes(reinterpret_cast<unsigned char*>(randomGarbage.data()),
               randomGarbage.size());

    QSqlQuery query(m_db.sqlDatabase());
    query.prepare(R"(
        UPDATE users
        SET encrypted_dek = :encrypted_dek,
            wallet_wiped = 1,
            wiped_at = :wiped_at
        WHERE id = :user_id
    )");

    query.bindValue(":encrypted_dek", randomGarbage);
    query.bindValue(":wiped_at", CryptoUtils::currentTimestampMs());
    query.bindValue(":user_id", userId);

    if (!query.exec()) {
        return Result<void>::fail(
            ErrorCode::DatabaseUpdateFailed,
            query.lastError().text()
        );
    }

    return Result<void>::ok();
}
```

这样做的意义：

```text
即使 wallets.encrypted_seed 没有被物理覆盖，由于 encrypted_dek 已被破坏，钱包种子也无法被解密。
```

---

### 6.3 WalletRepository 新增方法

```cpp
class WalletRepository {
public:
    Result<void> removeByUserId(const QString& userId);
};
```

实现示例：

```cpp
Result<void> WalletRepository::removeByUserId(const QString& userId)
{
    QSqlQuery query(m_db.sqlDatabase());
    query.prepare("DELETE FROM wallets WHERE user_id = :user_id");
    query.bindValue(":user_id", userId);

    if (!query.exec()) {
        return Result<void>::fail(
            ErrorCode::DatabaseDeleteFailed,
            query.lastError().text()
        );
    }

    return Result<void>::ok();
}
```

---

### 6.4 TransactionRepository 新增方法

```cpp
class TransactionRepository {
public:
    Result<void> removeByUserId(const QString& userId);
    Result<void> removeByWalletId(const QString& walletId);
};
```

推荐 SQL：

```sql
DELETE FROM transactions
WHERE wallet_id IN (
    SELECT id FROM wallets WHERE user_id = :user_id
);
```

注意顺序：

```text
先删除 transactions，再删除 wallets。
```

否则 wallets 删除后，可能无法再通过 user_id 找到 wallet_id。

---

## 7. 新增 SecurityService

建议新增服务：

```text
src/service/SecurityService.h
src/service/SecurityService.cpp
```

职责：

1. 钱包数据销毁；
2. 密钥销毁；
3. SQLite 安全清理；
4. 审计日志记录；
5. 自毁策略统一控制。

### 7.1 SecurityService.h

```cpp
class SecurityService : public QObject {
    Q_OBJECT

public:
    SecurityService(Database* db,
                    UserRepository* userRepo,
                    WalletRepository* walletRepo,
                    TransactionRepository* txRepo,
                    AuditLogRepository* auditRepo,
                    QObject* parent = nullptr);

    Result<void> wipeUserWalletData(const QString& userId,
                                    const QString& reason);

private:
    Database* m_db;
    UserRepository* m_userRepo;
    WalletRepository* m_walletRepo;
    TransactionRepository* m_txRepo;
    AuditLogRepository* m_auditRepo;

    Result<void> cleanupSQLite();
};
```

### 7.2 wipeUserWalletData 流程

```text
wipeUserWalletData(userId, reason)
  ↓
1. beginTransaction()
2. audit append "SECURITY_SELF_DESTRUCT"
3. destroyEncryptedDEK(userId)
4. remove transactions by userId
5. remove wallet by userId
6. markWalletWiped(userId)
7. commit()
8. PRAGMA wal_checkpoint(TRUNCATE)
9. PRAGMA secure_delete = ON
10. VACUUM
```

注意：

```text
审计日志最好在删除前写入，且不要删除 audit_logs。
```

### 7.3 示例实现

```cpp
Result<void> SecurityService::wipeUserWalletData(const QString& userId,
                                                 const QString& reason)
{
    auto tx = m_db->beginTransaction();
    if (tx.isFail()) {
        return tx;
    }

    QJsonObject details;
    details["reason"] = reason;
    details["policy"] = "password_failed_3_times";
    details["timestamp"] = QString::number(CryptoUtils::currentTimestampMs());

    auto auditRes = m_auditRepo->append(
        "SECURITY_SELF_DESTRUCT",
        userId,
        QString::fromUtf8(QJsonDocument(details).toJson(QJsonDocument::Compact))
    );

    if (auditRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(auditRes.error().code,
                                  auditRes.error().message);
    }

    auto destroyDekRes = m_userRepo->destroyEncryptedDEK(userId);
    if (destroyDekRes.isFail()) {
        m_db->rollback();
        return destroyDekRes;
    }

    auto removeTxRes = m_txRepo->removeByUserId(userId);
    if (removeTxRes.isFail()) {
        m_db->rollback();
        return removeTxRes;
    }

    auto removeWalletRes = m_walletRepo->removeByUserId(userId);
    if (removeWalletRes.isFail()) {
        m_db->rollback();
        return removeWalletRes;
    }

    auto markRes = m_userRepo->markWalletWiped(
        userId,
        CryptoUtils::currentTimestampMs()
    );

    if (markRes.isFail()) {
        m_db->rollback();
        return markRes;
    }

    auto commitRes = m_db->commit();
    if (commitRes.isFail()) {
        m_db->rollback();
        return commitRes;
    }

    cleanupSQLite();

    return Result<void>::ok();
}
```

---

## 8. AuthService 登录失败 3 次逻辑

### 8.1 新登录失败流程

```text
用户输入密码
  ↓
查找用户
  ↓
检查 wallet_wiped
  ↓
计算 password hash
  ↓
密码错误
  ↓
failed_attempts + 1
  ↓
如果 failed_attempts < 3
    返回：密码错误，剩余 N 次
  ↓
如果 failed_attempts >= 3
    调用 SecurityService::wipeUserWalletData()
    清空当前会话内存
    返回：密码错误 3 次，钱包数据已安全删除
```

### 8.2 AuthService 依赖增加

```cpp
class AuthService : public QObject {
public:
    AuthService(UserRepository* userRepo,
                AuditLogRepository* auditRepo,
                SecurityService* securityService,
                QThreadPool* cryptoPool,
                QObject* parent = nullptr);

private:
    SecurityService* m_securityService;
};
```

### 8.3 login 关键伪代码

```cpp
void AuthService::login(const QString& username, const QString& password)
{
    auto userResult = m_userRepo->findByUsername(username);
    if (userResult.isFail()) {
        emit loginCompleted(false, userResult.error().message);
        return;
    }

    auto optUser = userResult.value();
    if (!optUser.has_value()) {
        emit loginCompleted(false, "User not found");
        return;
    }

    User user = optUser.value();

    if (user.walletWiped) {
        emit loginCompleted(false,
            "Wallet data has been wiped. Please restore from encrypted backup.");
        return;
    }

    auto hashResult = KDFHelper::hashPassword(
        password,
        user.salt,
        user.kdfOpsLimit,
        user.kdfMemLimit
    );

    if (hashResult.isFail()) {
        emit loginCompleted(false, "Password verification failed");
        return;
    }

    bool passwordOk = CryptoUtils::secureCompare(
        hashResult.value(),
        SecureBuffer::fromQByteArray(user.passwordVerifier)
    );

    if (!passwordOk) {
        handleLoginFailure(user);
        return;
    }

    auto kekResult = KDFHelper::deriveKEK(
        password,
        user.salt,
        user.kdfOpsLimit,
        user.kdfMemLimit
    );

    if (kekResult.isFail()) {
        emit loginCompleted(false, "Failed to derive key");
        return;
    }

    auto dekResult = KeyManager::unwrapDEK(
        SecureBuffer::fromQByteArray(user.encryptedDEK),
        kekResult.value()
    );

    if (dekResult.isFail()) {
        emit loginCompleted(false, "Failed to unlock wallet");
        return;
    }

    m_currentUserId = user.id;
    m_currentUsername = user.username;
    m_currentDEK = std::move(dekResult.value());
    m_loggedIn = true;

    m_userRepo->resetFailedAttempts(user.id);
    m_userRepo->updateLoginTime(user.id, CryptoUtils::currentTimestampMs());

    m_auditRepo->append("LOGIN", user.id, "{}");

    emit loginCompleted(true, "");
}
```

### 8.4 handleLoginFailure

```cpp
void AuthService::handleLoginFailure(const User& user)
{
    auto incRes = m_userRepo->incrementFailedAttempts(user.id);
    if (incRes.isFail()) {
        emit loginCompleted(false, incRes.error().message);
        return;
    }

    auto refreshedUserRes = m_userRepo->findById(user.id);
    if (refreshedUserRes.isFail() || !refreshedUserRes.value().has_value()) {
        emit loginCompleted(false, "Failed to reload user state");
        return;
    }

    User refreshed = refreshedUserRes.value().value();

    QJsonObject details;
    details["failed_attempts"] = refreshed.failedAttempts;
    details["max_attempts"] = AppConstants::MAX_FAILED_ATTEMPTS;

    m_auditRepo->append(
        "LOGIN_FAILED",
        user.id,
        QString::fromUtf8(QJsonDocument(details).toJson(QJsonDocument::Compact))
    );

    if (refreshed.failedAttempts >= AppConstants::MAX_FAILED_ATTEMPTS) {
        auto wipeRes = m_securityService->wipeUserWalletData(
            user.id,
            "Password failed 3 times"
        );

        clearSessionSecrets();

        if (wipeRes.isFail()) {
            emit loginCompleted(false,
                "Password failed 3 times, but wallet wipe failed: " +
                wipeRes.error().message);
            return;
        }

        emit loginCompleted(false,
            "Password failed 3 times. Wallet data has been securely wiped.");
        return;
    }

    int remaining =
        AppConstants::MAX_FAILED_ATTEMPTS - refreshed.failedAttempts;

    emit loginCompleted(false,
        QString("Invalid password. %1 attempt(s) remaining before wallet wipe.")
        .arg(remaining));
}
```

---

## 9. 内存安全设计

失败 3 次时，不仅要删除数据库数据，还要清理内存。

### 9.1 AuthService 增加 clearSessionSecrets

```cpp
void AuthService::clearSessionSecrets()
{
    if (m_currentDEK.size() > 0) {
        m_currentDEK.clear();
    }

    m_currentUserId.clear();
    m_currentUsername.clear();
    m_loggedIn = false;

    if (m_sessionMonitor) {
        m_sessionMonitor->stop();
    }
}
```

### 9.2 SecureBuffer 要求

`SecureBuffer` 必须保证：

1. 构造时使用安全内存；
2. 禁止拷贝；
3. 允许移动；
4. 析构时自动清零；
5. `clear()` 时立即清零并释放。

示例：

```cpp
~SecureBuffer()
{
    if (m_data && m_size > 0) {
        OPENSSL_cleanse(m_data, m_size);
        OPENSSL_secure_free(m_data);
    }
}
```

如果 `OPENSSL_secure_malloc` 初始化失败，可以退化为：

```text
OPENSSL_malloc + OPENSSL_cleanse
```

但必须写入 warning 日志。

---

## 10. SQLite 安全删除策略

SQLite 普通 `DELETE` 不等于真正物理擦除。因此建议数据库启动时启用：

```sql
PRAGMA secure_delete = ON;
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
```

执行钱包擦除后：

```sql
PRAGMA wal_checkpoint(TRUNCATE);
VACUUM;
```

Database 增加方法：

```cpp
Result<void> Database::secureCleanup()
{
    auto r1 = execute("PRAGMA secure_delete = ON;");
    if (r1.isFail()) return r1;

    auto r2 = execute("PRAGMA wal_checkpoint(TRUNCATE);");
    if (r2.isFail()) return r2;

    auto r3 = execute("VACUUM;");
    if (r3.isFail()) return r3;

    return Result<void>::ok();
}
```

注意：

```text
VACUUM 不能在事务内部执行。
```

所以 `wipeUserWalletData()` 必须先 `commit()`，再调用 `secureCleanup()`。

---

## 11. UI 交互设计

### 11.1 登录失败提示

第 1 次密码错误：

```text
密码错误，还剩 2 次机会。
```

第 2 次密码错误：

```text
密码错误，还剩 1 次机会。再次错误将删除当前用户的钱包数据。
```

第 3 次密码错误：

```text
密码错误次数达到上限，当前用户的钱包数据已被安全删除。
请使用加密备份恢复钱包。
```

### 11.2 LoginViewModel 增加属性

```cpp
Q_PROPERTY(int failedAttempts READ failedAttempts NOTIFY failedAttemptsChanged)
Q_PROPERTY(int remainingAttempts READ remainingAttempts NOTIFY remainingAttemptsChanged)
Q_PROPERTY(bool walletWiped READ walletWiped NOTIFY walletWipedChanged)
Q_PROPERTY(QString securityWarning READ securityWarning NOTIFY securityWarningChanged)
```

### 11.3 LoginPage.qml 警告提示

```qml
Text {
    visible: loginViewModel.remainingAttempts === 1
    text: "警告：再次输入错误将删除当前用户的钱包数据"
    color: "#F44336"
    wrapMode: Text.WordWrap
}
```

---

## 12. 完整登录安全流程图

```text
┌──────────────────────┐
│ 用户输入用户名和密码   │
└──────────┬───────────┘
           │
           v
┌──────────────────────┐
│ findByUsername()      │
└──────────┬───────────┘
           │
   不存在  │  存在
     ┌─────┴──────┐
     v            v
返回用户不存在   检查 wallet_wiped
                  │
          已删除  │ 未删除
             ┌────┴────┐
             v         v
       提示需恢复    PBKDF2 校验密码
                       │
              错误     │ 正确
                ┌──────┴────────┐
                v               v
       failed_attempts + 1    解密 DEK
                │               │
         是否 >= 3？            v
          │        │         登录成功
         否        是
          │        │
          v        v
 提示剩余次数   SecurityService::wipeUserWalletData()
                   │
                   v
            清理 DEK / 钱包 / 交易
                   │
                   v
            写入审计日志
                   │
                   v
            提示钱包已删除
```

---

## 13. 模块结构调整

建议在现有结构基础上新增和调整以下文件：

```text
src/
├── service/
│   ├── AuthService.h/cpp
│   ├── SecurityService.h/cpp        # 新增：安全销毁、密钥擦除
│   ├── WalletService.h/cpp
│   ├── TransactionService.h/cpp
│   ├── BackupService.h/cpp
│   └── AuditService.h/cpp
├── persistence/
│   ├── UserRepository.h/cpp         # 新增 destroyEncryptedDEK / markWalletWiped
│   ├── WalletRepository.h/cpp       # 新增 removeByUserId
│   ├── TransactionRepository.h/cpp  # 新增 removeByUserId
│   ├── AuditLogRepository.h/cpp
│   └── Database.h/cpp               # 新增 secureCleanup
├── domain/
│   └── User.h                       # 新增 walletWiped / wipedAt
└── app/
    └── Constants.h                  # MAX_FAILED_ATTEMPTS = 3
```

---

## 14. main.cpp 依赖注入调整

新增 `SecurityService` 后，启动流程建议如下：

```cpp
auto userRepo = std::make_unique<UserRepository>(&database);
auto walletRepo = std::make_unique<WalletRepository>(&database);
auto txRepo = std::make_unique<TransactionRepository>(&database);
auto auditRepo = std::make_unique<AuditLogRepository>(&database);

auto securityService = std::make_unique<SecurityService>(
    &database,
    userRepo.get(),
    walletRepo.get(),
    txRepo.get(),
    auditRepo.get()
);

auto authService = std::make_unique<AuthService>(
    userRepo.get(),
    auditRepo.get(),
    securityService.get(),
    cryptoPool.get()
);

auto walletService = std::make_unique<WalletService>(
    walletRepo.get(),
    authService.get()
);

auto txService = std::make_unique<TransactionService>(
    walletRepo.get(),
    txRepo.get(),
    authService.get()
);
```

---

## 15. 错误码补充

建议在 `ErrorCode` 中增加 Security 系列：

```cpp
enum class ErrorCode {
    // Security 系列，800-899
    WalletWiped = 800,
    WalletWipeFailed = 801,
    MaxPasswordAttemptsExceeded = 802,
    SecurityPolicyViolation = 803,
    SecureCleanupFailed = 804,
};
```

错误消息建议：

```text
WalletWiped:
Wallet data has been wiped. Please restore from backup.

MaxPasswordAttemptsExceeded:
Maximum password attempts exceeded. Wallet data was securely deleted.

SecureCleanupFailed:
Wallet data was deleted, but SQLite secure cleanup failed.
```

---

## 16. 备份恢复策略

既然系统支持错误 3 次后删除钱包数据，就必须设计备份恢复闭环。

建议策略：

```text
注册成功后，强制提示用户创建备份。
未完成备份前，不允许大额转账。
错误 3 次删除钱包后，只能通过备份恢复。
```

### 16.1 users 表增加备份状态

```sql
backup_created INTEGER NOT NULL DEFAULT 0,
last_backup_at INTEGER NOT NULL DEFAULT 0
```

### 16.2 BackupService 导出成功后更新用户状态

```cpp
userRepo->markBackupCreated(userId, now);
auditRepo->append("BACKUP_EXPORT", userId, details);
```

### 16.3 钱包被擦除后的恢复流程

```text
1. 用户打开 RestorePage
2. 输入备份文件路径和备份密码
3. BackupService::importBackup()
4. 解密备份文件
5. 验证 format_id 和 integrity_hash
6. 恢复 users / wallets / transactions / audit_logs
7. 将 wallet_wiped 重新置为 0
8. 重新登录
```

---

## 17. MVP 实施阶段

### 第一阶段：本地单机钱包

1. 注册用户；
2. 登录用户；
3. 创建钱包；
4. 显示公钥地址；
5. 显示余额；
6. 模拟转账；
7. 登录错误 3 次删除当前用户钱包；
8. 审计日志记录。

### 第二阶段：备份恢复

1. 导出加密 JSON 备份；
2. 导入备份恢复用户和钱包；
3. 校验备份完整性。

### 第三阶段：增强安全

1. SecureBuffer；
2. SQLite secure_delete；
3. 审计日志哈希链；
4. 会话超时锁定；
5. 交易签名验签。

---

## 18. 最终安全策略总结

最终建议采用以下策略：

```text
1. 每个用户拥有独立 DEK。
2. 用户密码不直接加密钱包种子，而是派生 KEK。
3. KEK 只用于解密 encryptedDEK。
4. DEK 用于加密钱包种子 encryptedSeed。
5. 钱包种子用于生成 Ed25519 密钥对。
6. 密码错误 1 次：failed_attempts + 1。
7. 密码错误 2 次：强警告。
8. 密码错误 3 次：
   - 写入 SECURITY_SELF_DESTRUCT 审计日志；
   - 销毁 encryptedDEK；
   - 删除 wallets；
   - 删除 transactions；
   - 标记 users.wallet_wiped = 1；
   - 清理内存 DEK；
   - SQLite checkpoint + VACUUM；
9. 用户只能通过加密备份恢复钱包。
```

这套方案比单纯执行 `DELETE wallet` 更合理，因为它同时完成：

```text
业务删除 + 密钥销毁 + 审计记录 + 数据库清理 + UI 提示 + 备份恢复闭环
```

这才更接近一个真正硬件钱包系统的安全设计。

---

## 19. 实施优先级建议

| 优先级 | 任务 | 说明 |
|---|---|---|
| P0 | `MAX_FAILED_ATTEMPTS = 3` | 立即修改安全策略 |
| P0 | `users.wallet_wiped` 字段 | 标记钱包是否已销毁 |
| P0 | `SecurityService` | 统一执行安全销毁 |
| P0 | `destroyEncryptedDEK()` | 核心加密擦除能力 |
| P0 | `removeByUserId()` | 删除钱包和交易数据 |
| P1 | 审计日志扩展 | 记录 LOGIN_FAILED / SECURITY_SELF_DESTRUCT |
| P1 | UI 警告提示 | 第 2 次错误时强警告 |
| P1 | SQLite secure cleanup | checkpoint + VACUUM |
| P2 | 备份恢复闭环 | 支持钱包被删除后恢复 |
| P2 | 单元测试 | 测试 1/2/3 次错误流程 |

---

## 20. 推荐测试用例

### 20.1 密码错误 1 次

预期：

```text
failed_attempts = 1
wallet_wiped = 0
wallets 表仍存在数据
transactions 表仍存在数据
返回剩余 2 次提示
```

### 20.2 密码错误 2 次

预期：

```text
failed_attempts = 2
wallet_wiped = 0
返回强警告：再次错误将删除钱包数据
```

### 20.3 密码错误 3 次

预期：

```text
failed_attempts = 3
wallet_wiped = 1
encrypted_dek 被随机数据覆盖
wallets 表中当前用户钱包被删除
transactions 表中当前用户交易被删除
audit_logs 中存在 SECURITY_SELF_DESTRUCT
登录返回：钱包数据已安全删除
```

### 20.4 钱包删除后再次登录

预期：

```text
即使密码正确，也不允许进入钱包
提示：钱包数据已被删除，请通过备份恢复
```

### 20.5 备份恢复

预期：

```text
导入备份成功
wallet_wiped = 0
wallets 数据恢复
用户可以重新登录
```

