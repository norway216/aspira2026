# GrapeOS HP 音量默认值固化修改流程

## GrapeOS U 盘安装镜像中固化 HP 音量默认值完整修改流程

**适用镜像：** `GrapeOS-UDisk-Flasher-V0.0.0.25.img`

本文档整理了从映射原始镜像、修改 `roota.tar.gz`、写入首次启动默认音量脚本、重新打包、更新 `md5`、卸载映射、烧录验证的完整流程。

---

## 一、修改目标

| 项目 | 说明 |
|---|---|
| 默认行为 | 第一次安装或第一次启动系统时，自动设置 HP 音量为 `34`。 |
| 后续行为 | 用户后续如果把 HP 改成 `30` 并保存，下次重启继续保持 `30`。 |
| 关键原则 | 不要每次启动都强制执行 `HP=34`，而是只在首次启动时初始化默认值。 |
| 修改对象 | 外层镜像中的 `roota.tar.gz`，而不是 `roota.img`。 |
| 校验要求 | 替换 `roota.tar.gz` 后必须重新生成 `roota.tar.gz.md5`，并验证全部 `*.md5`。 |

---

## 二、准备工作

建议不要直接修改原始镜像，而是复制一份新镜像进行编辑。

```bash
mkdir -p ~/grapeos_patch
cd ~/grapeos_patch
cp /home/xxx/GrapeOS-UDisk-Flasher-V0.0.0.25.img ./GrapeOS-UDisk-Flasher-V0.0.0.25.img.bak
cp /home/xxx/GrapeOS-UDisk-Flasher-V0.0.0.25.img ./GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img
```

---

## 三、映射镜像分区

使用 `kpartx` 将镜像中的分区映射到 `/dev/mapper` 下。后续示例假设外层安装分区是 `/dev/mapper/loop9p2`，实际操作时要以你的 `lsblk` 输出为准。

```bash
sudo apt update
sudo apt install -y kpartx util-linux
cd ~/grapeos_patch
sudo kpartx -av GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img
lsblk
ls /dev/mapper/
```

---

## 四、挂载外层安装分区

挂载后看到的不是最终系统根目录，而是 U 盘安装/恢复镜像的外层目录。

```bash
sudo mkdir -p /mnt/rootfs
sudo mount /dev/mapper/loop9p2 /mnt/rootfs
ls -lh /mnt/rootfs
```

---

## 五、确认真正的 rootfs 包

外层目录中通常会有 `boot.img`、`data.tar.gz`、`roota.img`、`roota.tar.gz`、`version.ini` 等文件。之前已经验证 `roota.img` 虽然是 ext4，但挂载后为空，真正的系统 rootfs 是 `roota.tar.gz`。

```bash
tar -tzf /mnt/rootfs/roota.tar.gz | head -50
```

---

## 六、不要读写挂载 roota.img

不要用读写方式挂载 `roota.img`。即使不写文件，ext4 读写挂载也可能更新 journal、mount count、last mounted time 等元数据，导致 `roota.img.md5` 校验失败。只查看时必须使用 `ro` 方式。

```bash
sudo mkdir -p /mnt/roota
sudo mount -o loop,ro /mnt/rootfs/roota.img /mnt/roota
ls /mnt/roota
sudo umount /mnt/roota
```

---

## 七、解压 roota.tar.gz

将真正 rootfs 解压到宿主机工作目录。解压时如果出现 `aic_load_fw.ko`、`aic8800_fdrv.ko` 时间戳在 2073 年的警告，一般只是原始打包环境时间异常，不是解压失败。

```bash
mkdir -p ~/grapeos_patch/roota_extract
cd ~/grapeos_patch/roota_extract
sudo tar -xpf /mnt/rootfs/roota.tar.gz
ls
```

---

## 八、确认 amixer 与 alsactl 路径

脚本里会直接使用绝对路径，因此需要先确认目标 rootfs 中的实际路径。

```bash
cd ~/grapeos_patch/roota_extract
find . -name amixer
find . -name alsactl
```

---

## 九、写入“首次初始化 HP=34，后续不覆盖”的脚本

这是本次修改的核心。脚本逻辑是：如果初始化标记不存在，说明是第一次启动，设置 `HP=34` 并保存 ALSA 状态；如果标记已经存在，则不再强制改成 `34`，只恢复上次保存的 ALSA 状态。

```bash
cd ~/grapeos_patch/roota_extract
sudo mkdir -p usr/local/sbin
sudo tee usr/local/sbin/set-hp-volume.sh > /dev/null <<'EOF'
#!/bin/sh

MARKER="/var/lib/chison/hp-volume-initialized"

mkdir -p /var/lib/chison

# 第一次启动：设置默认 HP=34
if [ ! -f "$MARKER" ]; then
    sleep 5

    /usr/bin/amixer -c 0 sset 'HPVOL' on 2>/dev/null || true
    /usr/bin/amixer -c 0 sset 'HP' 34

    sleep 5

    /usr/bin/amixer -c 0 sset 'HPVOL' on 2>/dev/null || true
    /usr/bin/amixer -c 0 sset 'HP' 34

    /usr/sbin/alsactl store 2>/dev/null || true

    date > "$MARKER"
    exit 0
fi

# 后续启动：恢复用户上次保存的 ALSA 状态，不再强制改成 34
/usr/sbin/alsactl restore 2>/dev/null || true

exit 0
EOF
sudo chmod +x usr/local/sbin/set-hp-volume.sh
```

---

## 十、写入 SysV init 启动脚本

系统中存在 `/etc/rc2.d`、`/etc/rc3.d` 等目录，因此使用 SysV init 方式最稳。该脚本在启动时调用音量初始化脚本，在关机或重启时执行 `alsactl store` 保存用户当前设置。

```bash
sudo tee etc/init.d/set-hp-volume > /dev/null <<'EOF'
#!/bin/sh
### BEGIN INIT INFO
# Provides:          set-hp-volume
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Initialize HP volume once and save ALSA state
### END INIT INFO

case "$1" in
    start)
        /usr/local/sbin/set-hp-volume.sh &
        ;;
    stop)
        /usr/sbin/alsactl store 2>/dev/null || true
        ;;
    restart|force-reload)
        /usr/sbin/alsactl store 2>/dev/null || true
        /usr/local/sbin/set-hp-volume.sh &
        ;;
    *)
        echo "Usage: /etc/init.d/set-hp-volume {start|stop|restart|force-reload}"
        exit 1
        ;;
esac

exit 0
EOF
sudo chmod +x etc/init.d/set-hp-volume
```

---

## 十一、创建启动与停止链接

`S99` 表示在正常运行级别中尽量靠后启动，避免被 ALSA、PulseAudio、LightDM 等服务覆盖。`K01` 表示关机或重启时尽早保存当前 ALSA 状态。

```bash
sudo ln -sf ../init.d/set-hp-volume etc/rc2.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc3.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc4.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc5.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc0.d/K01set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc1.d/K01set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc6.d/K01set-hp-volume
```

---

## 十二、确认写入成功

```bash
ls -l usr/local/sbin/set-hp-volume.sh
ls -l etc/init.d/set-hp-volume
ls -l etc/rc2.d/S99set-hp-volume
ls -l etc/rc3.d/S99set-hp-volume
ls -l etc/rc4.d/S99set-hp-volume
ls -l etc/rc5.d/S99set-hp-volume
ls -l etc/rc0.d/K01set-hp-volume
ls -l etc/rc1.d/K01set-hp-volume
ls -l etc/rc6.d/K01set-hp-volume
```

---

## 十三、可选：修正未来时间戳警告

如果解压时出现 2073 年时间戳警告，可以顺手修正对应驱动文件的时间戳。

```bash
cd ~/grapeos_patch/roota_extract
sudo touch modules/aic_load_fw.ko 2>/dev/null || true
sudo touch modules/aic8800_fdrv.ko 2>/dev/null || true
```

---

## 十四、重新打包 roota.tar.gz

重新打包时需要在 `roota_extract` 目录中执行，使用 `.` 作为打包根路径，以保持 `./bin`、`./etc` 这类路径格式。

```bash
cd ~/grapeos_patch/roota_extract
sudo tar --numeric-owner -czpf ../roota.tar.gz .
ls -lh ~/grapeos_patch/roota.tar.gz
```

---

## 十五、备份原始 roota.tar.gz 到宿主机

不要把备份文件放到 `/mnt/rootfs` 里面，因为镜像分区空间有限，可能无法再容纳一个 1.5G 的备份包。

```bash
mkdir -p ~/grapeos_patch/backup
cp /mnt/rootfs/roota.tar.gz ~/grapeos_patch/backup/roota.tar.gz.bak
cp /mnt/rootfs/roota.tar.gz.md5 ~/grapeos_patch/backup/roota.tar.gz.md5.bak
cat ~/grapeos_patch/backup/roota.tar.gz.md5.bak
```

---

## 十六、替换外层镜像里的 roota.tar.gz

由于外层分区空间有限，通常不能同时放新旧两个 `roota.tar.gz`，因此先删除旧包，再复制新包。

```bash
df -h /mnt/rootfs
ls -lh /mnt/rootfs/roota.tar.gz
sudo rm -f /mnt/rootfs/roota.tar.gz
sync
sudo cp ~/grapeos_patch/roota.tar.gz /mnt/rootfs/roota.tar.gz
sync
ls -lh /mnt/rootfs/roota.tar.gz
```

---

## 十七、重新生成 roota.tar.gz.md5

必须保持原始 md5 文件格式，即“md5 值 + 两个空格 + 文件名”。不要只写纯 md5 值，否则 `md5sum -c` 可能无法校验。

```bash
cd /mnt/rootfs
sudo md5sum roota.tar.gz | sudo tee roota.tar.gz.md5
sync
cat roota.tar.gz.md5
```

---

## 十八、验证全部 md5

这一步非常重要。之前曾经因为读写挂载 `roota.img` 导致 `roota.img.md5` 失败。最终必须确保所有 md5 都成功。

```bash
cd /mnt/rootfs
for f in *.md5; do
    echo "Checking $f"
    md5sum -c "$f" || echo "FAILED: $f"
done

# 如果 roota.img.md5 失败，说明 roota.img 曾经被读写挂载过，可以重新生成：
# sudo md5sum roota.img | sudo tee roota.img.md5
# sync
# 然后重新执行全部 md5 校验。
```

---

## 十九、检查外层关键文件是否还在

安装界面依赖外层关键文件，不能缺失。

```bash
ls -lh /mnt/rootfs/boot.img
ls -lh /mnt/rootfs/initrd.img
ls -lh /mnt/rootfs/recovery.img
ls -lh /mnt/rootfs/flash.bin
ls -lh /mnt/rootfs/kernela.img
ls -lh /mnt/rootfs/roota.tar.gz
ls -lh /mnt/rootfs/roota.tar.gz.md5
ls -lh /mnt/rootfs/version.ini
```

---

## 二十、卸载外层分区

卸载前必须退出 `/mnt/rootfs` 目录并执行 `sync`。如果提示 `busy`，通常是终端还停留在挂载目录。

```bash
cd ~
sync
sudo umount /mnt/rootfs
mount | grep /mnt/rootfs

# 如果提示 busy：
# sudo lsof +f -- /mnt/rootfs
# sudo fuser -vm /mnt/rootfs
```

---

## 二十一、释放 kpartx 映射

卸载完成后释放镜像分区映射。

```bash
sudo kpartx -dv ~/grapeos_patch/GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img
lsblk
ls /dev/mapper/
```

---

## 二十二、烧录到 U 盘

确认 U 盘设备名后，使用整盘设备进行烧录，例如 `/dev/sdb`，不要写成 `/dev/sdb1`。

```bash
lsblk
sudo dd if=~/grapeos_patch/GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img of=/dev/sdb bs=4M status=progress conv=fsync
sync
```

---

## 二十三、RK3588 安装测试

将 U 盘插入 RK3588 开发板，按原有触发方式进入安装界面：开机长按 3~4 秒，再短按 1~2 秒三次，进入安装界面后选择 `install system`。

如果不能进入安装界面，优先重新挂载镜像检查全部 `*.md5`，尤其确认 `roota.img` 和 `roota.tar.gz` 都是“成功”。

---

## 二十四、安装后验证默认值

系统安装完成并进入 UltrasoundOS 后，验证 HP 是否默认为 `34`。

```bash
amixer -c 0 get 'HP'
ls -l /usr/local/sbin/set-hp-volume.sh
ls -l /etc/init.d/set-hp-volume
ls -l /etc/rc2.d/S99set-hp-volume
ls -l /etc/rc6.d/K01set-hp-volume
ls -l /var/lib/chison/hp-volume-initialized
cat /var/lib/chison/hp-volume-initialized
```

---

## 二十五、验证“用户修改后保持”的效果

手动把 HP 改成 `30` 并保存，然后重启验证是否保持 `30`。

```bash
amixer -c 0 sset 'HP' 30
alsactl store
amixer -c 0 get 'HP'
reboot

# 重启后再次执行：
amixer -c 0 get 'HP'
```

---

## 二十六、关键注意事项

- 本流程修改的是 `GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img` 中的 `roota.tar.gz`。
- `roota.img` 不要读写挂载；如果必须查看，请使用 `mount -o loop,ro`。
- 替换 `roota.tar.gz` 后必须重新生成 `roota.tar.gz.md5`，并执行 `md5sum -c` 验证。
- 全部 `*.md5` 都成功后再卸载并释放 `kpartx` 映射。
- 如果用户直接断电，关机阶段的 `alsactl store` 可能来不及执行；最稳方式是用户修改音量后手动执行 `alsactl store`。
- 如果重启后仍被改回 `34`，重点检查 `/var/lib/chison/hp-volume-initialized` 是否存在，以及脚本是否仍是“每次强制设置 34”的旧版本。

---

# 附录：完整命令汇总版

```bash
# 0. 准备工作
mkdir -p ~/grapeos_patch
cd ~/grapeos_patch
cp /home/xxx/GrapeOS-UDisk-Flasher-V0.0.0.25.img \
    ./GrapeOS-UDisk-Flasher-V0.0.0.25.img.bak
cp /home/xxx/GrapeOS-UDisk-Flasher-V0.0.0.25.img \
    ./GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img

# 1. 映射镜像
sudo apt update
sudo apt install -y kpartx util-linux
sudo kpartx -av GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img
lsblk
ls /dev/mapper/

# 2. 挂载外层分区，按实际情况替换 loop9p2
sudo mkdir -p /mnt/rootfs
sudo mount /dev/mapper/loop9p2 /mnt/rootfs
ls -lh /mnt/rootfs

# 3. 确认 roota.tar.gz 是 rootfs
tar -tzf /mnt/rootfs/roota.tar.gz | head -50

# 4. 解压 roota.tar.gz
mkdir -p ~/grapeos_patch/roota_extract
cd ~/grapeos_patch/roota_extract
sudo tar -xpf /mnt/rootfs/roota.tar.gz

# 5. 确认工具路径
find . -name amixer
find . -name alsactl

# 6. 写入首次初始化脚本
sudo mkdir -p usr/local/sbin
sudo tee usr/local/sbin/set-hp-volume.sh > /dev/null <<'EOF'
#!/bin/sh
MARKER="/var/lib/chison/hp-volume-initialized"
mkdir -p /var/lib/chison
if [ ! -f "$MARKER" ]; then
    sleep 5
    /usr/bin/amixer -c 0 sset 'HPVOL' on 2>/dev/null || true
    /usr/bin/amixer -c 0 sset 'HP' 34
    sleep 5
    /usr/bin/amixer -c 0 sset 'HPVOL' on 2>/dev/null || true
    /usr/bin/amixer -c 0 sset 'HP' 34
    /usr/sbin/alsactl store 2>/dev/null || true
    date > "$MARKER"
    exit 0
fi
/usr/sbin/alsactl restore 2>/dev/null || true
exit 0
EOF
sudo chmod +x usr/local/sbin/set-hp-volume.sh

# 7. 写入 SysV init 脚本
sudo tee etc/init.d/set-hp-volume > /dev/null <<'EOF'
#!/bin/sh
### BEGIN INIT INFO
# Provides:          set-hp-volume
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Initialize HP volume once and save ALSA state
### END INIT INFO
case "$1" in
    start)
        /usr/local/sbin/set-hp-volume.sh &
        ;;
    stop)
        /usr/sbin/alsactl store 2>/dev/null || true
        ;;
    restart|force-reload)
        /usr/sbin/alsactl store 2>/dev/null || true
        /usr/local/sbin/set-hp-volume.sh &
        ;;
    *)
        echo "Usage: /etc/init.d/set-hp-volume {start|stop|restart|force-reload}"
        exit 1
        ;;
esac
exit 0
EOF
sudo chmod +x etc/init.d/set-hp-volume

# 8. 创建启动/停止链接
sudo ln -sf ../init.d/set-hp-volume etc/rc2.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc3.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc4.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc5.d/S99set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc0.d/K01set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc1.d/K01set-hp-volume
sudo ln -sf ../init.d/set-hp-volume etc/rc6.d/K01set-hp-volume

# 9. 重新打包
cd ~/grapeos_patch/roota_extract
sudo tar --numeric-owner -czpf ../roota.tar.gz .

# 10. 备份原包到宿主机
mkdir -p ~/grapeos_patch/backup
cp /mnt/rootfs/roota.tar.gz ~/grapeos_patch/backup/roota.tar.gz.bak
cp /mnt/rootfs/roota.tar.gz.md5 ~/grapeos_patch/backup/roota.tar.gz.md5.bak

# 11. 替换外层镜像中的 roota.tar.gz
sudo rm -f /mnt/rootfs/roota.tar.gz
sync
sudo cp ~/grapeos_patch/roota.tar.gz /mnt/rootfs/roota.tar.gz
sync

# 12. 重新生成 md5 并验证全部 md5
cd /mnt/rootfs
sudo md5sum roota.tar.gz | sudo tee roota.tar.gz.md5
sync
for f in *.md5; do
    echo "Checking $f"
    md5sum -c "$f" || echo "FAILED: $f"
done

# 如果 roota.img.md5 失败：
# sudo md5sum roota.img | sudo tee roota.img.md5
# sync
# 然后重新执行全部 md5 校验。

# 13. 卸载并释放映射
cd ~
sync
sudo umount /mnt/rootfs
sudo kpartx -dv ~/grapeos_patch/GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img

# 14. 烧录，按实际 U 盘设备替换 /dev/sdb
sudo dd if=~/grapeos_patch/GrapeOS-UDisk-Flasher-V0.0.0.25-hp34.img \
    of=/dev/sdb \
    bs=4M \
    status=progress \
    conv=fsync
sync
```
