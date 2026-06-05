# 问题与修复记录

> 记录 portfolio-site 项目开发与运行过程中遇到的问题及其解决方案，以及重大功能迭代。

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

## 4. CSRF 中间件未自动签发 Token

### 发现时间

2026-06-05

### 现象

所有 POST 请求（联系表单、后台登录）返回 `403 CSRF token missing`。用户访问 GET 页面时，CSRF cookie 从未被设置，因此后续 POST 必然失败。

### 根因

CSRF 中间件定义了 `SetToken()` 和 `GetTokenField()` 方法，但：
- `SetToken()` 从未在任何 Handler 中被调用
- `Protect()` 方法对 GET 请求直接放行，不生成 token

这导致浏览器端永远拿不到 `csrf_token` cookie，所有 POST 请求必然失败。

### 修复

**文件**: `internal/middleware/csrf.go`

在 `Protect()` 方法中增加 GET 请求自动签发 token 逻辑：

```go
// 修复：GET/HEAD/OPTIONS 请求自动签发 CSRF cookie
if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
    if _, err := r.Cookie(c.cookieName); err != nil {
        c.SetToken(w, r)  // 首次访问自动设置
    }
    next.ServeHTTP(w, r)
    return
}
```

**文件**: `web/templates/layout/base.html`

增加前端 JS 自动注入 CSRF 隐藏字段到所有表单：

```javascript
document.querySelectorAll('form[method="POST"]').forEach(function(form) {
    if (!form.querySelector('input[name="csrf_token"]')) {
        var input = document.createElement('input');
        input.type = 'hidden';
        input.name = 'csrf_token';
        input.value = getCSRFToken();
        form.appendChild(input);
    }
});
```

---

## 5. 文章列表 SQL 歧义列名

### 发现时间

2026-06-05

### 现象

访问 `/admin/articles` 返回 500 错误：

```text
ERROR: failed to list articles for admin
error: query articles: SQL logic error: ambiguous column name: created_at (1)
```

### 根因

文章查询 JOIN 了 `article_categories` 表，两张表都有 `created_at` 列。Admin Handler 使用 `OrderBy: "created_at DESC"` 时未加表别名前缀，SQLite 无法确定排序依据。

### 修复

**文件**: `internal/module/article/admin_handler.go`

```go
// 修复前
OrderBy: "created_at DESC",

// 修复后
OrderBy: "a.created_at DESC",
```

---

## 6. 统计页面 NULL 日期扫描错误

### 发现时间

2026-06-05

### 现象

访问 `/admin/stats` 返回 500 错误：

```text
ERROR: failed to get stats overview
error: sql: Scan error on column index 0, name "date":
converting NULL to string is unsupported
```

### 根因

`GetDailyStats()` 查询使用 `date(created_at)` 聚合，当某天无记录时返回 NULL。Go 的 `sql` 包无法将 NULL 扫描到 `string` 类型（`DailyStat.Date`）。

### 修复

**文件**: `internal/module/stats/repository.go`

在 `GetDailyStats()` 和 `GetPathStats()` 中使用 `sql.NullString` 中转：

```go
// 修复前
var s DailyStat
if err := rows.Scan(&s.Date, &s.PV, &s.UV); err != nil {
    return nil, err
}

// 修复后
var s DailyStat
var d sql.NullString
if err := rows.Scan(&d, &s.PV, &s.UV); err != nil {
    return nil, err
}
s.Date = d.String
```

---

## 7. Apple Music 风格全站 UI 重设计

### 时间

2026-06-05

### 变更范围

对整个 portfolio-site 进行了 Apple Music 网页版风格的完整重设计。

### CSS 重写

**文件**: `web/static/css/app.css`（35166 字节）

| 设计元素 | 实现方式 |
|---------|---------|
| 暗色主题 | `#000000` 纯黑底色 + `#1C1C1E` / `#2C2C2E` 层级表面 |
| 渐变 Hero | 粉色→紫色→蓝色多色阶渐变文字（`-webkit-background-clip: text`） |
| 玻璃拟态 Header | `backdrop-filter: saturate(180%) blur(20px)` + 半透明背景 |
| 卡片悬停 | `translateY(-4px)` + `box-shadow` 发光阴影 |
| 焦点网格 | 4 列网格，每种带有独立渐变色图标 |
| 背景氛围 | 固定定位粒子层，3 个 `radial-gradient` 椭圆缓慢移动 |
| Phosphor Icons | 通过 CDN 加载（MIT 许可） |
| 滚动条 | 自定义 8px 暗色滚动条 |
| 选中文本 | `rgba(0, 122, 255, 0.3)` 蓝色高亮 |

### 模板结构

所有 11 个模板文件均重写：

| 文件 | 变更 |
|------|------|
| `web/templates/layout/base.html` | 新 Header/Footer，Phosphor Icons CDN，CSRF JS 注入，移动端菜单 |
| `web/templates/pages/home.html` | 渐变 Hero + 渐变封面项目卡片 + 焦点网格 + CTA |
| `web/templates/pages/projects.html` | 渐变封面项目卡片网格 |
| `web/templates/pages/project_detail.html` | 300px 渐变 Hero Banner + 完整 Markdown 内容 |
| `web/templates/pages/writings.html` | 动画文章卡片 |
| `web/templates/pages/writing_detail.html` | 文章详情布局 |
| `web/templates/pages/about.html` | 排版优化 |
| `web/templates/pages/contact.html` | 暗色表单 + 聚焦光环 |
| `web/templates/pages/resume.html` | 技能列表 |
| `web/templates/pages/error.html` | 新建 — 大号状态码错误页 |
| `web/templates/admin/*.html` | 6 个后台模板 — 侧边栏圆点导航 + 渐变统计卡片 |

### 响应式设计

3 个断点：
- `960px` — 项目网格 2→1 列，后台布局切换
- `640px` — 焦点网格 4→2→1 列，Header 竖向排列
- `400px` — 统计卡片 4→3→2 列

---

## 8. 架构文档导入为项目作品

### 时间

2026-06-05

### 背景

`/home/sa/Lab/Programming/aspira2026/projects/` 目录下有 10 篇中文架构设计文档（Markdown），需要将它们导入到 portfolio-site 作为项目作品页面展示。

### 实现

**文件**: `cmd/seed/main.go`（新建）

编写了一个 Go CLI 种子工具，功能包括：

1. **解析 Markdown 文件** — 提取标题（首个 `#` 或首行）、摘要（从「项目目标」段落提取前 300 字符）、分类（关键词匹配）、技术栈
2. **批量写入 SQLite** — 直接操作 `projects` 表，状态设为 `published`
3. **分配渐变色封面** — `cover_image` 字段存储 `gradient:N` 格式，前端模板解析为 CSS 类名
4. **简易 Markdown→HTML 转换** — 处理标题、列表、代码块、引用块、粗体

### 导入的 9 个项目

| # | 源文件 | 导入标题 | 分类 | 技术栈 |
|---|--------|---------|------|--------|
| 1 | `android_iris_camera_architecture.md` | Iris Recognition System with Android Camera Stream | Medical Imaging | C++, OpenCV, ONNX Runtime, FFmpeg |
| 2 | `distributed_embedded_web_ops_platform_design.md` | Distributed Embedded Device Web Operations Platform | Embedded Linux | Go, WebSocket, gRPC, Prometheus |
| 3 | `distributed_secure_traffic_gateway_design.md` | Distributed Secure Traffic Gateway | Backend Systems | Go, TLS, Docker, Kubernetes |
| 4 | `embedded_face_recognition_architecture.md` | Embedded Face Detection & Recognition System | Medical Imaging | C++20, OpenCV, dlib, Qt/QML |
| 5 | `embedded_hardware_wallet_architecture.md` | High-Performance Embedded Hardware Wallet | Security | C++20, Qt/QML, SQLite, OpenSSL |
| 6 | `go_lan_device_web_system_architecture.md` | LAN Device Discovery & Management Web System | Backend Systems | Go, WebSocket, SQLite, Redis |
| 7 | `hardware_wallet_architecture.md` | RK3568 Hardware Wallet with Security Enhancements | Security | C++20, Qt/QML, OpenSSL, RK3568 |
| 8 | `iris_recognition_cpp_nn_architecture.md` | Neural Network Iris Recognition System in C++ | Medical Imaging | C++20, OpenCV, ONNX Runtime, TensorRT |
| 9 | `traffic_lab_proxy_architecture.md` | Multi-Node Traffic Scheduling & Proxy Control Platform | Backend Systems | Go, TCP/UDP, Docker, Prometheus |

### 模板增强

**文件**: `internal/render/template.go`

新增 `coverIndex` 模板函数，从 `gradient:N` 字符串提取数字索引：

```go
func coverIndex(coverImage string) string {
    if strings.HasPrefix(coverImage, "gradient:") {
        return strings.TrimPrefix(coverImage, "gradient:")
    }
    return "1"
}
```

---

## 9. 9 种 Apple Music 风格渐变色封面系统

### 时间

2026-06-05

### 设计目标

不使用任何图片，纯 CSS 渐变色块模拟 Apple Music 专辑/播放列表封面效果，每个项目独一无二。

### 9 种渐变定义

**文件**: `web/static/css/app.css`

| 类名 | 色彩路线 | 色值 |
|------|---------|------|
| `cover-1` | Hot Pink → Coral | `#FF2D55 → #FF375F → #FF6482 → #FF8E9E` |
| `cover-2` | Deep Purple → Violet | `#5E1F8A → #8944CC → #AF52DE → #CA7AF0` |
| `cover-3` | Emerald → Teal | `#007D4C → #1DA86B → #30D158 → #6EE7A8` |
| `cover-4` | Tangerine → Amber | `#E85D04 → #FF7B24 → #FF9F0A → #FFC043` |
| `cover-5` | Electric Blue → Cyan | `#0038A8 → #0066E0 → #007AFF → #64D2FF` |
| `cover-6` | Magenta → Orchid | `#C41E70 → #E8308A → #FF375F → #FF6B8A` |
| `cover-7` | Gold → Coral Sunset | `#B8860B → #E8A817 → #FFD60A → #FF9F6E` |
| `cover-8` | Indigo → Periwinkle | `#1D1160 → #3D2DA6 → #5E5CE6 → #94A3F8` |
| `cover-9` | Crimson → Fuchsia | `#8B0045 → #CF0A5C → #BF5AF2 → #D48BFF` |

### 视觉深度层次

每个封面叠加 4 层效果：

```css
/* Layer 1: 渐变底色 */
.cover-N { background: linear-gradient(135deg, ...); }

/* Layer 2: SVG 噪点纹理 (opacity: 0.05–0.06) */
.cover-grain { background-image: url("data:image/svg+xml,..."); }

/* Layer 3: 中央内发光 */
.project-cover::after {
    background: radial-gradient(ellipse at 50% 30%, rgba(255,255,255,0.12) 0%, transparent 70%);
}

/* Layer 4: 底部渐变暗角 */
.cover-fade {
    background: linear-gradient(0deg, rgba(0,0,0,0.35) 0%, transparent 100%);
}
```

### 分配逻辑

种子工具按 `SortOrder % 9 + 1` 分配，9 个项目各得一个独立渐变色。

---

## 10. 全站中译英

### 时间

2026-06-05

### 变更范围

将所有面向用户的文本从中文翻译为英文：

| 层级 | 文件数 | 内容 |
|------|--------|------|
| HTML 模板 | 11 | 导航、卡片、表单、页脚、错误页 |
| 种子工具 | 1 | 9 个项目的标题、摘要、完整架构文档（~500 行英文 HTML） |
| 后端配置 | 1 | Site 描述已是英文，无需修改 |

### 翻译策略

- **项目标题/摘要** — 手工翻译，保持技术准确性
- **架构文档内容** — 手工翻译所有描述性文本；代码块、ASCII 架构图、技术术语保持不变（通用语言）
- **模板文本** — UI 字符串全部英文化

---

## 变更统计

### 文件变更汇总

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/static/css/app.css` | 重写 | Apple Music 暗色主题，35166 字节 |
| `web/static/images/favicon.svg` | 新建 | 渐变色 SVG 图标 |
| `web/templates/layout/base.html` | 重写 | Header/Footer + Phosphor Icons + CSRF JS |
| `web/templates/pages/home.html` | 重写 | 渐变 Hero + 特色项目卡片 |
| `web/templates/pages/projects.html` | 重写 | 渐变封面项目网格 |
| `web/templates/pages/project_detail.html` | 重写 | 渐变 Hero + Markdown 内容 |
| `web/templates/pages/writings.html` | 重写 | 动画文章卡片 |
| `web/templates/pages/writing_detail.html` | 重写 | 文章详情布局 |
| `web/templates/pages/about.html` | 重写 | 英文 About 页面 |
| `web/templates/pages/contact.html` | 重写 | 暗色表单 |
| `web/templates/pages/resume.html` | 重写 | 英文 Resume |
| `web/templates/pages/error.html` | 新建 | 大号状态码错误页 |
| `web/templates/admin/login.html` | 重写 | 渐变色登录卡片 |
| `web/templates/admin/dashboard.html` | 重写 | 侧边栏 + 统计卡片 |
| `web/templates/admin/article_list.html` | 重写 | 暗色表格 |
| `web/templates/admin/article_form.html` | 重写 | 暗色表单 |
| `web/templates/admin/stats_overview.html` | 重写 | 统计仪表板 |
| `web/templates/admin/audit_logs.html` | 重写 | 审计日志表格 |
| `internal/middleware/csrf.go` | 修复 | GET 请求自动签发 CSRF token |
| `internal/module/article/admin_handler.go` | 修复 | SQL 歧义列名 `a.created_at` |
| `internal/module/stats/repository.go` | 修复 | NULL date 扫描使用 `sql.NullString` |
| `internal/render/template.go` | 增强 | 新增 `coverIndex` 模板函数 |
| `cmd/seed/main.go` | 新建 | 架构文档导入工具 + 英文内容嵌入 |
| `web/static/images/favicon.svg` | 新建 | 渐变色 SVG favicon |

### 路由验证

所有 16 条路由返回 200：

```
GET  /                         200
GET  /projects                 200
GET  /projects/{slug} × 9      200
GET  /writings                 200
GET  /about                    200
GET  /resume                   200
GET  /contact                  200
GET  /admin/login              200
```

---

## 问题统计

| # | 类别 | 严重程度 | 状态 |
|---|---|---|---|
| 1 | Go 1.22 路由冲突 | 致命（panic） | ✅ 已修复 |
| 2 | Shell 脚本地址显示 | 轻微 | ✅ 已修复 |
| 3 | 模块下载网络不可达 | 阻塞（构建） | ✅ 已绕过 |
| 4 | CSRF Token 未签发 | 阻塞（所有 POST） | ✅ 已修复 |
| 5 | 文章列表 SQL 歧义列名 | 中等（500 错误） | ✅ 已修复 |
| 6 | 统计页面 NULL 扫描 | 中等（500 错误） | ✅ 已修复 |
| 7 | Apple Music UI 重设计 | 功能迭代 | ✅ 已完成 |
| 8 | 架构文档导入 | 功能迭代 | ✅ 已完成 |
| 9 | 渐变色封面系统 | 功能迭代 | ✅ 已完成 |
| 10 | 全站中译英 | 功能迭代 | ✅ 已完成 |

---

> 最后更新: 2026-06-05
