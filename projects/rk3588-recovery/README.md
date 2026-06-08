# RK3588 Recovery System v2.0

系统安装 / 备份 / 还原 & 软件备份还原系统

基于 RK3588 平台的高性能、高稳定性 Recovery 系统。

## 架构

```
┌─────────────────────────────────┐
│  TUI Interface (ANSI Terminal)  │  ← 菜单导航、进度条、日志查看
├─────────────────────────────────┤
│  Service Layer (C++)            │
│  ┌───────────┐ ┌──────────────┐ │
│  │ Install   │ │ Backup/Restore│ │  ← dd/xz/tar 引擎
│  │ Engine    │ │ Engine        │ │
│  ├───────────┤ ├──────────────┤ │
│  │ Partition │ │ Checksum     │ │  ← 分区管理、SHA256校验
│  │ Manager   │ │ Verifier     │ │
│  ├───────────┤ ├──────────────┤ │
│  │ Logger    │ │ GPIO Monitor │ │  ← 操作日志、电源键检测
│  └───────────┘ └──────────────┘ │
├─────────────────────────────────┤
│  Core Layer (C)                 │
│  Block I/O (4MB dd) · xz/tar   │  ← 高性能磁盘操作
│  MD5/SHA256 · GPIO sysfs       │
└─────────────────────────────────┘
```

## 功能

| 功能 | 实现 |
|---|---|
| 系统安装 | xzcat + dd 烧写 system 分区镜像 |
| 系统备份 | dd + xz -9 完整备份 system 分区 |
| 系统还原 | 从备份镜像恢复 system 分区 |
| 软件备份 | tar.xz 管理 /home 或 /userdata/apps |
| 软件还原 | 从 tar.xz 备份恢复应用数据 |
| 镜像校验 | SHA256 / MD5 校验 |
| 操作日志 | 带时间戳的完整操作记录 |
| GPIO 检测 | 长按 3 秒进入 Recovery 模式 |

## 编译

```bash
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
make -j$(nproc)
```

## 运行

```bash
# 模拟器模式（使用文件模拟分区）
./rk3588_recovery

# 真实硬件模式
./rk3588_recovery --real
```

## 性能

- 4MB 块 I/O（dd 级别效率）
- 多线程进度报告
- OpenSSL 硬件加速 SHA256
- xz -9 最高压缩率
- 零外部 UI 依赖（纯 ANSI 终端）

## 分区布局

| 分区 | 大小 | 用途 |
|---|---|---|
| boot | 64MB | Bootloader |
| system | 8-16GB | 系统 rootfs |
| recovery | 512MB | Recovery 系统 |
| userdata | 16-32GB | 用户数据/备份 |
