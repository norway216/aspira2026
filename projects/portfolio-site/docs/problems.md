# 问题与修复记录

> 记录 portfolio-site 项目开发与运行过程中遇到的问题及其解决方案。

---

## 1. Go 1.22 ServeMux 路由冲突

### 发现时间

2026-06-05

### 现象

```text
panic: pattern "GET /" conflicts with pattern "/static/":
GET / matches fewer methods than /static/, but has a more general path pattern
```

服务启动时直接 panic，无法运行。

### 根因

Go 1.22 对 `net/http.ServeMux` 进行了重大增强，支持了 `METHOD /path` 格式的模式匹配。新模式引入了更严格的冲突检测机制：

- `GET /` 只匹配 GET 方法的根路径，但路径模式较宽泛（`/` 是 `/static/...` 的前缀）
- `/static/` 匹配所有 HTTP 方法，虽然路径更精确（限定在 `/static/` 子树下），但方法限定更宽泛

Go 1.22 的路由器无法判断哪个模式优先级更高，因此在注册阶段直接 panic。

### 修复

**文件**: `internal/app/router.go`

两处改动：

1. 首页路由使用精确匹配后缀 `{$}`：

```go
// 修复前（会 panic）
mux.HandleFunc("GET /", a.PageHandler.Home)

// 修复后（{$} 表示仅匹配精确路径 /，不包含子路径）
mux.HandleFunc("GET /{$}", a.PageHandler.Home)
```

2. 静态文件加上方法限定：

```go
// 修复前
mux.Handle("/static/", http.StripPrefix("/static/", fs))

// 修复后
mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
mux.Handle("HEAD /static/", http.StripPrefix("/static/", fs))
```

### 相关知识

Go 1.22 ServeMux 模式匹配规则：

| 模式 | 说明 |
|---|---|
| `GET /` | 过时，会与子路径模式冲突 |
| `GET /{$}` | 精确匹配根路径 |
| `/` | 通配所有路径（不含方法限定） |
| `GET /items/{id}` | 路径参数 |
| `GET /static/` | 匹配 `/static/` 及其所有子路径 |

### 参考

- [Go 1.22 Release Notes — Enhanced routing](https://go.dev/doc/go1.22#enhanced_routing)

---

## 2. run.sh 地址显示缺少冒号

### 发现时间

2026-06-05

### 现象

```text
[INFO]  地址: http://localhost8080
```

端口号前缺少 `:`，显示为 `localhost8080` 而非 `localhost:8080`。

### 根因

`run.sh` 中使用了 `${SERVER_ADDR#*:}` 参数展开来提取端口号，但拼接 URL 时缺少冒号：

```bash
# 错误写法：剥离了冒号但拼接时未补回
echo "http://localhost${SERVER_ADDR#*:}"

# SERVER_ADDR=:8080 → ${SERVER_ADDR#*:}=8080 → http://localhost8080
```

### 修复

**文件**: `run.sh`

改为直接使用 `${SERVER_ADDR}`，因为它本身就包含冒号：

```bash
# 修复后
local host="${SERVER_ADDR}"
echo "http://localhost${host}"
# → http://localhost:8080
```

涉及位置：
- `start_server()` 函数中的启动信息输出
- `dev_mode()` 函数中的开发模式提示
- `show_status()` 函数中的状态显示和健康检查 curl 命令

---

## 3. go mod tidy 网络不可达

### 发现时间

2026-06-05

### 现象

```text
dial tcp 173.194.43.141:443: connect: connection refused
```

执行 `go mod tidy` 时无法连接 Go 模块代理。

### 根因

开发环境无外网访问能力，无法连接到 `proxy.golang.org` 和 `sum.golang.org`。

### 修复

使用本地 Go 模块缓存，并跳过校验和验证：

```bash
GONOSUMCHECK='*' GONOSUMDB='*' \
    GOPROXY="file://${GOPATH}/pkg/mod/cache/download" \
    go mod download
```

**文件**: `go.mod`

手动指定已缓存模块的精确版本：

```
require (
    golang.org/x/crypto v0.28.0
    modernc.org/sqlite v1.33.1
)
```

注意：`modernc.org/sqlite` v1.51.0 要求 Go >= 1.25.0，当前环境 Go 1.22.2 不兼容，因此降级到 v1.33.1（要求 Go 1.20）。

### 补充问题：缺少 .info 文件

部分缓存模块缺失 `.info` 元数据文件（仅有 `.mod`、`.zip`、`.ziphash`），需手动创建：

```bash
echo '{"Version":"v1.55.3"}' > $GOPATH/pkg/mod/cache/download/modernc.org/libc/@v/v1.55.3.info
echo '{"Version":"v1.8.0"}'  > $GOPATH/pkg/mod/cache/download/modernc.org/memory/@v/v1.8.0.info
echo '{"Version":"v1.2.0"}'  > $GOPATH/pkg/mod/cache/download/modernc.org/strutil/@v/v1.2.0.info
echo '{"Version":"v1.1.0"}'  > $GOPATH/pkg/mod/cache/download/modernc.org/token/@v/v1.1.0.info
```

---

## 问题统计

| # | 类别 | 严重程度 | 状态 |
|---|---|---|---|
| 1 | Go 1.22 路由冲突 | 致命（panic） | ✅ 已修复 |
| 2 | Shell 脚本地址显示 | 轻微 | ✅ 已修复 |
| 3 | 模块下载网络不可达 | 阻塞（构建） | ✅ 已绕过 |

---

> 最后更新: 2026-06-05
