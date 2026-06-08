# RK3588 系统安装 / 备份 / 还原 & 软件备份还原架构设计

## 一、设计目标

- 长按电源键 3 秒进入 Recovery 系统界面
- Recovery 软件独立于系统镜像，可独立升级
- 系统镜像、软件、用户数据分离
- 支持：
  - 系统安装
  - 系统备份/还原
  - 软件备份/还原
- 安全、可靠，可记录日志并校验数据
- 可扩展支持 USB/SD 卡或 OTA 更新

## 二、RK3588 分区规划

| 分区 | 类型 | 功能 |
|------|------|------|
| boot | FAT32 (64MB) | Bootloader (U-Boot/TrustBoot) |
| system | EXT4 (8~16GB) | 正常系统 rootfs |
| recovery | EXT4 (512MB) | Recovery 系统 + 备份工具 |
| userdata | EXT4 (16~32GB) | 用户数据、备份镜像、软件更新包 |

> Tip: 通过 recovery 分区可以独立操作 system 分区或 userdata 分区，保证系统升级或备份不影响 Recovery 软件。

## 三、电源键触发流程

```
上电/重启
     |
     v
Bootloader 检测 GPIO 电源键
     |
  +--------------------+
  | 长按 >= 3 秒        | -> 设置 bootmode = recovery -> 启动 recovery 分区
  | 短按 / 无按         | -> 正常启动 system 分区
  +--------------------+
```

- Bootloader 捕获电源键长按事件
- 设置环境变量 `bootcmd_recovery` 启动 recovery 分区
- 正常启动不加载 recovery 软件

## 四、Recovery 系统架构

### 1. UI 层

- QtQuick/QML 或 Framebuffer / TUI
- 功能菜单：

```
1. Install System
2. Backup System
3. Restore System
4. Backup Apps
5. Restore Apps
6. Exit / Reboot
```

- 显示进度条、日志、操作提示

### 2. 服务层 (Service Layer)

| 模块 | 功能 |
|------|------|
| 系统安装 | dd + xz 烧写 system 分区镜像 |
| 系统备份/还原 | dd + xz 完整备份与恢复 system 分区 |
| 软件备份/还原 | tar + xz 管理 /home 或 /userdata/apps |
| 存储管理 | 挂载/卸载分区、检测剩余空间 |
| 日志 & 校验 | md5/sha256 校验、操作日志记录 |

### 3. 核心驱动层

- GPIO / 电源键检测
- I/O 访问（eMMC / NAND / USB / SD）
- 分区挂载
- 任务调度与安全校验

## 五、软件模块设计

### 系统安装

```bash
umount /dev/mmcblk0p2
xzcat system.img.xz | dd of=/dev/mmcblk0p2 bs=4M status=progress
sync
```

### 系统备份

```bash
mkdir -p /userdata/backup/system
dd if=/dev/mmcblk0p2 of=/userdata/backup/system/system_$(date +%F).img bs=4M status=progress
xz -9 /userdata/backup/system/system_$(date +%F).img
```

### 系统还原

```bash
xzcat /userdata/backup/system/system_YYYY-MM-DD.img.xz | dd of=/dev/mmcblk0p2 bs=4M status=progress
sync
```

### 软件备份

```bash
tar -cJf /userdata/backup/apps_YYYY-MM-DD.tar.xz /home /userdata/apps
```

### 软件还原

```bash
tar -xJf /userdata/backup/apps_YYYY-MM-DD.tar.xz -C /
```

- 支持多版本备份与选择性恢复

## 六、安全与鲁棒性

- 镜像校验 (md5/sha256)
- 操作确认提示，防止误操作
- 日志记录操作时间、用户、结果
- 软件升级独立于 system 分区
- 支持回滚到上一次有效系统

## 七、软件更新与维护

- Recovery 软件存放在 `recovery` 分区 + `/userdata/backup_tools`
- 更新方式：
  1. 插入 U 盘，自动覆盖 `/userdata/backup_tools`
  2. OTA / 网络下载更新
- 优点：
  - 软件与系统分离
  - 系统升级不影响功能
  - 可独立维护

## 八、操作流程示意图

```
[按电源 3 秒]
         |
         v
[Bootloader 检测 GPIO]
         |
     长按 -> 进入 recovery 分区
     短按 -> 正常启动 system
         |
         v
+---------------------------+
| Recovery Menu             |
| 1. Install System         |
| 2. Backup System          |
| 3. Restore System         |
| 4. Backup Apps            |
| 5. Restore Apps           |
| 6. Exit / Reboot          |
+---------------------------+
         |
         v
[Service Layer: dd/xz/tar/挂载/日志/校验]
         |
         v
[Storage: system / userdata / recovery 分区]
         |
         v
[完成提示 / 日志 / 重启]
```

## 九、方案优势

- 系统软件分离，独立升级
- 安全可靠，支持校验与日志
- 用户数据与系统分区分离，防止误操作
- 支持 U 盘/SD/OTA 更新
- 可扩展，适用于 RK3588 平台和类似嵌入式系统
