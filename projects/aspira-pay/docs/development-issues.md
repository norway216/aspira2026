# Aspira Pay V2 — 开发问题记录

> 记录项目开发过程中遇到的技术问题及解决方案，便于后续维护和排查。

---

## 1. 一键部署脚本：宿主机无 psql 导致数据库初始化失败

### 日期

2026-06-07

### 现象

```
[WARN]  PostgreSQL not reachable at localhost:5432
[INFO]  Attempting to start via Docker...
[INFO]  PostgreSQL Docker container started
[ERROR] PostgreSQL not available — cannot initialize database
```

Docker 容器实际已启动运行，但脚本报告数据库不可达。

### 根因

脚本使用 `pg_isready` 和 `psql` 命令检测/操作数据库，但宿主机未安装 PostgreSQL 客户端工具。即使容器内数据库已就绪，宿主机也无法通过 `localhost:5432` 建立客户端连接（这些命令不存在）。

### 解决方案

新增 4 个辅助函数，自动降级到 `docker exec`：

```bash
# 数据库就绪检测 — 自动选择本地命令或 docker exec
_pg_isready() {
    if command -v pg_isready &> /dev/null; then
        pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME"
    elif docker exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# 执行 SQL 文件 — docker cp 进容器执行
_pg_exec_file() {
    local file="$1"
    if command -v psql &> /dev/null; then
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file"
    else
        docker cp "$file" "$CONTAINER_NAME":/tmp/migration.sql
        docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_NAME" \
            psql -U "$DB_USER" -d "$DB_NAME" -f /tmp/migration.sql
        docker exec "$CONTAINER_NAME" rm -f /tmp/migration.sql
    fi
}

# 执行 SQL 语句
_pg_exec() {
    local sql="$1"
    if command -v psql &> /dev/null; then
        psql -h "$DB_HOST" ... -c "$sql"
    else
        docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_NAME" \
            psql -U "$DB_USER" -d "$DB_NAME" -c "$sql"
    fi
}

# 轮询等待数据库就绪（最多 30 秒）
_wait_for_postgres() {
    local max_wait=30
    while [ $waited -lt $max_wait ]; do
        if _pg_isready; then return 0; fi
        sleep 1
    done
    return 1
}
```

### 影响范围

- `check_infra()` — 用 `_pg_isready` + `_wait_for_postgres` 替代固定 `sleep 3`
- `db_init()` — 用 `_pg_exec_file` 替代 `psql -f`
- `db_reset()` — 用 `_pg_exec` 替代 `psql -c`
- `status()` — 用 `_pg_isready` 替代 `pg_isready`

---

## 2. C++ 引擎编译失败：对象不可拷贝

### 日期

2026-06-07

### 现象

```text
error: use of deleted function 'aspira::engine::WAL& aspira::engine::WAL::operator=(const aspira::engine::WAL&)'
note: 'aspira::engine::WAL& aspira::engine::WAL::operator=(const aspira::engine::WAL&)' is implicitly deleted
because the default definition would be ill-formed
note: use of deleted function 'std::basic_fstream<...>::operator=(const std::basic_fstream<...>&)'
note: use of deleted function 'std::mutex& std::mutex::operator=(const std::mutex&)'
```

### 根因

`Engine` 类中将 `WAL` 声明为值类型成员：

```cpp
// Engine.h (before)
WAL wal_{"engine.wal"};
```

`WAL` 类内部包含两个**不可拷贝、不可赋值**的成员：

| 类型 | 为何不可拷贝 |
|---|---|
| `std::fstream file_` | C++11 起 `std::fstream` 的拷贝构造和拷贝赋值被标记为 `= delete` |
| `std::mutex mutex_` | `std::mutex` 的拷贝构造和拷贝赋值被标记为 `= delete` |

当 `Engine::init()` 中执行 `wal_ = WAL(wal_path)` 时，编译器尝试调用 `WAL::operator=`，但该函数因为成员不可拷贝而被隐式删除。

### 解决方案

改用 `std::unique_ptr<WAL>` 延迟构造：

```cpp
// Engine.h (after)
std::unique_ptr<WAL> wal_;

// Engine.cpp — init()
wal_ = std::make_unique<WAL>(wal_path);

// 使用时
if (wal_) wal_->log_command(cmd);   // wal_.xxx() → wal_->xxx()
```

`std::unique_ptr` 只持有指针，不要求对象本身可拷贝。对象在 `make_unique` 时构造，生命周期由 unique_ptr 管理。

### 关键经验

C++ 中以下标准库类型**不可拷贝/赋值**，当它们作为类成员出现时会传染性地让包含类也不可拷贝：

- `std::fstream`, `std::ifstream`, `std::ofstream`
- `std::mutex`, `std::recursive_mutex`, `std::shared_mutex`
- `std::unique_ptr<T>`（但它是**可移动的**）
- `std::thread`

设计原则：
1. 包含这些成员的类如果需要"重新初始化"行为，用 `std::unique_ptr` 包装
2. 或用 `std::optional` (C++17) + `emplace()`
3. 或确保在构造函数初始化列表中完整构造，不提供重新赋值路径

---

## 3. C++ 引擎编译失败：缺失 #include 头文件

### 日期

2026-06-07

### 现象

多个编译错误，分 4 处：

```text
// 1. Ledger.cpp
error: 'unique_lock' is not a member of 'std'

// 2. WAL.h
error: 'vector' in namespace 'std' does not name a template type

// 3. Publisher.h
error: 'atomic' in namespace 'std' does not name a template type

// 4. Engine.cpp
error: 'sprintf' was not declared in this scope
```

### 根因

各自缺少对应的标准库头文件包含：

| 文件 | 缺少的 include | 原因 |
|---|---|---|
| `Ledger.cpp` | `<mutex>` | 使用了 `std::unique_lock`，但只间接包含了 `<shared_mutex>` |
| `WAL.h` | `<vector>` | 声明 `std::vector<Entry> read_all()` 返回值 |
| `Publisher.h` | `<atomic>` | 声明 `std::atomic<uint64_t> total_published_` 成员 |
| `Engine.cpp` | `<cstdio>` | 使用了 `sprintf()` 函数 |

### 解决方案

分别在对应文件中添加缺失的 `#include`：

```cpp
// Ledger.cpp
#include <mutex>

// WAL.h
#include <vector>

// Publisher.h
#include <atomic>

// Engine.cpp
#include <cstdio>
```

### 关键经验

C++17/20 标准库的头文件包含关系更精确，不再像 C++14 以前那样"宽包含"：

- `<shared_mutex>` 不一定包含 `<mutex>` → 不能依赖间接包含 `std::unique_lock`
- `<functional>` 不一定包含 `<vector>` → 不能依赖间接包含容器
- `<string>` 不一定包含 `<cstdio>` → C 函数必须显式包含 C 兼容头文件

最佳实践：**每个源文件显式包含它直接使用的所有标准库头文件**。

---

## 4. C++ 引擎编译失败：PaymentCommand 缺少 command_type 字段

### 日期

2026-06-07

### 现象

```text
error: 'const struct aspira::engine::PaymentCommand' has no member named 'command_type'
   94 |     switch (cmd.command_type) {
```

### 根因

`Types.h` 中 `PaymentCommand` 结构体原本没有 `command_type` 字段，但 `Engine.cpp` 的 `process_command()` 方法需要通过 `cmd.command_type` 来判断执行哪个处理逻辑（FREEZE / EXECUTE / RELEASE / REFUND）。

这是架构设计时协议定义和业务逻辑未对齐的典型问题。

### 解决方案

在 `PaymentCommand` 中添加 `command_type` 字段，并提供默认值：

```cpp
struct PaymentCommand {
    // ... existing fields ...
    CommandType command_type = CommandType::EXECUTE_PAYMENT;  // 新增
    std::string from_account;
    // ...
};
```

默认值 `EXECUTE_PAYMENT` 确保向后兼容——旧代码构造的 `PaymentCommand` 会直接走执行路径。

---

## 总结

| 问题 | 类型 | 修复方式 |
|---|---|---|
| 宿主机无 psql | Shell 脚本兼容性 | `docker exec` 降级方案 + 轮询等待 |
| WAL 对象不可拷贝 | C++ 类设计 | `unique_ptr<WAL>` 延迟构造 |
| 缺失 include 头文件 | C++ 编译规范 | 显式添加 `<mutex>`, `<vector>`, `<atomic>`, `<cstdio>` |
| PaymentCommand 缺字段 | 协议/业务对齐 | 添加 `command_type` 字段 + 默认值 |
