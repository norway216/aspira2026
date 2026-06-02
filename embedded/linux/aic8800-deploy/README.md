# AIC8800 USB WiFi 驱动部署指南

## 概述

- **芯片**: AIC/Aicsemi AIC8800D80 系列
- **USB VID:PID**: 
  - 存储模式: `a69c:5721` (MSC)
  - WiFi 模式: `a69c:8D80`
- **内核**: 针对 Linux 6.6.63 (ARM64 JelinaOS) 交叉编译
- **驱动源码**: [Wolfdv1/aic8800](https://github.com/Wolfdv1/aic8800) (已修补支持 PID 0x8D80)

## 部署步骤

### 1. 将文件传输到目标设备

```bash
# 在开发机上打包
cd /home/sa/Lab/arm_linux/arm_debian/aic8800-deploy
tar czf aic8800-deploy.tar.gz *
# 将 aic8800-deploy.tar.gz 传输到目标设备
```

### 2. 在目标设备上安装

```bash
# 在目标 ARM64 设备上
tar xzf aic8800-deploy.tar.gz
cd scripts/
chmod +x install.sh
sudo ./install.sh
```

### 3. 插入 USB WiFi 适配器

插入适配器后：

#### 自动方式（如果 eject 失败）
```bash
# 手动触发模式切换
sudo usb_modeswitch -KQ -v 0a69c -p 5721
```

#### 检查驱动是否加载
```bash
lsmod | grep aic
# 应该看到: aic8800_fdrv, aic_load_fw

dmesg | tail -20
# 查看驱动初始化日志
```

### 4. 验证 WiFi 功能

```bash
# 查看无线网卡
iwconfig
# 或
iw dev

# 查看网络接口
ip link

# 扫描 WiFi
sudo iw dev wlan0 scan | grep SSID

# 连接 WiFi (使用 wpa_supplicant)
sudo wpa_passphrase "YOUR_SSID" "YOUR_PASSWORD" > /etc/wpa_supplicant.conf
sudo wpa_supplicant -B -i wlan0 -c /etc/wpa_supplicant.conf
sudo dhclient wlan0

# 或使用 NetworkManager
sudo nmcli device wifi list
sudo nmcli device wifi connect "YOUR_SSID" password "YOUR_PASSWORD"
```

## 文件说明

| 文件 | 说明 |
|---|---|
| `modules/aic_load_fw.ko` | 固件加载模块 (必须先加载) |
| `modules/aic8800_fdrv.ko` | 主 WiFi 驱动模块 |
| `firmware/aic8800DC/` | 固件文件 (部署到 /lib/firmware/aic8800DC/) |
| `udev/99-aic8800.rules` | udev 规则 (自动模式切换 + 驱动加载) |
| `scripts/install.sh` | 一键安装脚本 |

## 故障排查

### 问题: 插入后只有存储设备，没有 WiFi
```bash
# 确认 USB 模式
lsusb | grep a69c
# 如果显示 5721，说明还在存储模式

# 手动切换
sudo eject /dev/sdX    # 替换 sdX 为实际设备名
# 或
sudo usb_modeswitch -KQ -v 0a69c -p 5721

# 再次检查
lsusb | grep a69c
# 应该显示 8d80
```

### 问题: 驱动加载失败
```bash
dmesg | grep -i aic
dmesg | grep -i error
# 确认固件路径正确
ls /lib/firmware/aic8800DC/
```

### 问题: 驱动加载了但无法扫描
```bash
# 确认 rfkill 没有阻止
rfkill list
sudo rfkill unblock all

# 确认接口已启用
sudo ip link set wlan0 up
```

## 重新编译（如果需要）

```bash
# 修改驱动源码后重新编译
cd /home/sa/Lab/arm_linux/arm_debian/aic8800-driver/drivers/aic8800
CONFIG_SHELL=bash make \
  KDIR=/home/sa/Lab/arm_linux/arm_debian/kernel/linux-6.6.63 \
  ARCH=arm64 \
  CROSS_COMPILE=aarch64-linux-gnu- \
  KVER=6.6.63 \
  KBUILD_MODPOST_WARN=1 \
  clean modules
```
