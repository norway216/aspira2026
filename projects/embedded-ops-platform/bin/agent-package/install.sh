#!/bin/sh
# Embedded Agent Install Script
set -e

ARCH=$(uname -m)
case "$ARCH" in
  aarch64) BIN="embedded-agent-arm64" ;;
  armv7l)  BIN="embedded-agent-arm32" ;;
  x86_64)  BIN="embedded-agent" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

echo "Installing Embedded Agent for $ARCH..."

install -m 0755 bin/$BIN /usr/local/bin/embedded-agent
install -d /etc/embedded-agent /var/lib/embedded-agent /var/log
install -m 0644 config/config.yaml /etc/embedded-agent/config.yaml
install -m 0644 systemd/embedded-agent.service /etc/systemd/system/

systemctl daemon-reload
systemctl enable embedded-agent
systemctl start embedded-agent

echo "Agent installed and started. Check status with: systemctl status embedded-agent"
