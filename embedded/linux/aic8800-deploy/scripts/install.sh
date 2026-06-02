#!/bin/sh
# AIC8800 USB WiFi Driver Installation Script
# Target: ARM64 Linux kernel 6.6.63 (JelinaOS)
# Usage: Run as root on the target device
#        chmod +x install.sh && sudo ./install.sh

set -e

KVER=$(uname -r)
MODULE_DIR="/lib/modules/${KVER}/kernel/drivers/net/wireless/aic8800"
FW_DIR="/lib/firmware/aic8800DC"

echo "=== AIC8800 USB WiFi Driver Install ==="
echo "Kernel version: ${KVER}"

# 1. Install firmware
echo "[1/5] Installing firmware to ${FW_DIR}..."
mkdir -p "${FW_DIR}"
cp -v ../firmware/aic8800DC/*.bin "${FW_DIR}/"
cp -v ../firmware/aic8800DC/*.txt "${FW_DIR}/"
echo "Firmware installed."

# 2. Install kernel modules
echo "[2/5] Installing kernel modules to ${MODULE_DIR}..."
mkdir -p "${MODULE_DIR}"
cp -v ../modules/aic_load_fw.ko "${MODULE_DIR}/"
cp -v ../modules/aic8800_fdrv.ko "${MODULE_DIR}/"

# 3. Run depmod
echo "[3/5] Running depmod..."
/sbin/depmod -a ${KVER}

# 4. Install udev rules for USB mode switch
echo "[4/5] Installing udev rules..."
if [ -f ../udev/99-aic8800.rules ]; then
    cp -v ../udev/99-aic8800.rules /etc/udev/rules.d/
    udevadm control --reload-rules 2>/dev/null || true
    udevadm trigger 2>/dev/null || true
fi

# 5. Load modules
echo "[5/5] Loading modules..."
modprobe aic_load_fw 2>/dev/null || insmod "${MODULE_DIR}/aic_load_fw.ko"
modprobe aic8800_fdrv 2>/dev/null || insmod "${MODULE_DIR}/aic8800_fdrv.ko"

echo ""
echo "=== Installation complete ==="
echo "Plug in your AIC8800 USB WiFi adapter."
echo ""
echo "If the adapter is already plugged in, run:"
echo "  sudo usb_modeswitch -KQ -v 0a69c -p 5721"
echo ""
echo "Then check:"
echo "  lsmod | grep aic"
echo "  iwconfig"
echo "  ip link"
