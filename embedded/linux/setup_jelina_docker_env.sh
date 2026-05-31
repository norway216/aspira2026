#!/usr/bin/env bash
# ==============================================================================
# Jelina BSP / Yocto Docker Build Environment Setup
# Target example:
#   JelinaBSP-rk-xiangsheng-v1.2.2-2026-02-10T16-22-09
#
# What this script does on the HOST machine:
#   1. Creates a Debian 11 based Docker image for Jelina BSP / Yocto build.
#   2. Installs common Yocto dependencies:
#      chrpath, diffstat, gawk, zstd, uidmap, locales, etc.
#   3. Generates en_US.UTF-8 locale.
#   4. Forces /bin/sh -> /bin/bash inside the container image.
#   5. Optionally disables Ubuntu 24.04 AppArmor user namespace restriction.
#   6. Starts a privileged Docker container suitable for BitBake.
#
# Run on host:
#   chmod +x setup_jelina_docker_env.sh
#   ./setup_jelina_docker_env.sh
#
# Then inside the container:
#   cd ~/workspace/JelinaBSP-rk-xiangsheng-v1.2.2-2026-02-10T16-22-09
#   source setup-environment
#   bitbake jelina-flasher-image
# ==============================================================================

set -euo pipefail

# -----------------------------
# User configurable variables
# -----------------------------
IMAGE_NAME="${IMAGE_NAME:-jelina-builder:debian11}"
CONTAINER_NAME="${CONTAINER_NAME:-jelina-builder-debian11}"

# Host directories. Adjust these if your paths are different.
HOST_BASE_DIR="${HOST_BASE_DIR:-$HOME/arm-linux-build}"
HOST_WORKSPACE="${HOST_WORKSPACE:-$HOST_BASE_DIR/workspace}"
HOST_DOWNLOADS="${HOST_DOWNLOADS:-$HOST_BASE_DIR/downloads}"
HOST_OUTPUT="${HOST_OUTPUT:-$HOST_BASE_DIR/output}"
HOST_ROOTFS="${HOST_ROOTFS:-$HOST_BASE_DIR/rootfs}"
HOST_SCRIPTS="${HOST_SCRIPTS:-$HOST_BASE_DIR/scripts}"

# Docker build context directory.
DOCKER_CONTEXT="${DOCKER_CONTEXT:-$HOST_BASE_DIR/jelina-docker}"

# Set to 1 if you want this script to change the host sysctl automatically.
# Ubuntu 24.04 may restrict unprivileged user namespaces via AppArmor.
FIX_HOST_USERNS="${FIX_HOST_USERNS:-1}"

# Set to 1 if you want the sysctl change to be persistent after reboot.
PERSIST_HOST_USERNS="${PERSIST_HOST_USERNS:-0}"

# -----------------------------
# Helper functions
# -----------------------------
log() {
    echo
    echo "========== $* =========="
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "ERROR: command not found: $1"
        echo "Please install it first."
        exit 1
    fi
}

# -----------------------------
# 0. Check host requirements
# -----------------------------
log "Checking host requirements"
need_cmd docker

if ! docker info >/dev/null 2>&1; then
    echo "ERROR: Docker daemon is not available or current user has no permission."
    echo "Try:"
    echo "  sudo systemctl start docker"
    echo "  sudo usermod -aG docker \$USER"
    echo "Then re-login."
    exit 1
fi

# -----------------------------
# 1. Create host directories
# -----------------------------
log "Creating host directories"
mkdir -p "$HOST_WORKSPACE" "$HOST_DOWNLOADS" "$HOST_OUTPUT" "$HOST_ROOTFS" "$HOST_SCRIPTS" "$DOCKER_CONTEXT"

echo "HOST_WORKSPACE = $HOST_WORKSPACE"
echo "HOST_DOWNLOADS = $HOST_DOWNLOADS"
echo "HOST_OUTPUT    = $HOST_OUTPUT"
echo "DOCKER_CONTEXT = $DOCKER_CONTEXT"

# -----------------------------
# 2. Ubuntu 24.04 AppArmor user namespace fix
# -----------------------------
log "Checking host AppArmor user namespace restriction"

if [[ -r /proc/sys/kernel/apparmor_restrict_unprivileged_userns ]]; then
    current_userns_restrict="$(cat /proc/sys/kernel/apparmor_restrict_unprivileged_userns || true)"
    echo "kernel.apparmor_restrict_unprivileged_userns = $current_userns_restrict"

    if [[ "$current_userns_restrict" != "0" ]]; then
        if [[ "$FIX_HOST_USERNS" == "1" ]]; then
            echo "Trying to disable AppArmor unprivileged user namespace restriction on host..."
            sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0

            if [[ "$PERSIST_HOST_USERNS" == "1" ]]; then
                echo "Persisting sysctl setting to /etc/sysctl.d/99-jelina-userns.conf"
                echo 'kernel.apparmor_restrict_unprivileged_userns=0' | sudo tee /etc/sysctl.d/99-jelina-userns.conf >/dev/null
                sudo sysctl --system >/dev/null
            fi
        else
            echo "WARNING: User namespace restriction is enabled."
            echo "BitBake may fail with:"
            echo "  User namespaces are not usable by BitBake, possibly due to AppArmor."
            echo "To fix manually on host:"
            echo "  sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0"
        fi
    fi
else
    echo "No /proc/sys/kernel/apparmor_restrict_unprivileged_userns found."
    echo "This host may not use Ubuntu 24.04 style AppArmor user namespace restriction."
fi

# -----------------------------
# 3. Create Dockerfile
# -----------------------------
log "Generating Dockerfile"

cat > "$DOCKER_CONTEXT/Dockerfile" <<'EOF'
FROM debian:11

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8
ENV LANGUAGE=en_US:en

RUN apt-get update && apt-get install -y --no-install-recommends \
    sudo \
    bash \
    locales \
    ca-certificates \
    git \
    repo \
    wget \
    curl \
    gawk \
    diffstat \
    unzip \
    texinfo \
    gcc \
    g++ \
    build-essential \
    chrpath \
    socat \
    cpio \
    python3 \
    python3-pip \
    python3-pexpect \
    python3-git \
    python3-jinja2 \
    python3-subunit \
    xz-utils \
    debianutils \
    iputils-ping \
    file \
    zstd \
    liblz4-tool \
    uidmap \
    xterm \
    vim \
    nano \
    less \
    procps \
    util-linux \
    && sed -i 's/^# *en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen \
    && locale-gen en_US.UTF-8 \
    && update-locale LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8 \
    && ln -sf /bin/bash /bin/sh \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create builder user.
RUN useradd -m -s /bin/bash builder \
    && echo "builder ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/builder \
    && chmod 0440 /etc/sudoers.d/builder

USER builder
WORKDIR /home/builder

# Persistent shell environment.
RUN echo 'export LANG=en_US.UTF-8' >> /home/builder/.bashrc \
    && echo 'export LC_ALL=en_US.UTF-8' >> /home/builder/.bashrc \
    && echo 'export LANGUAGE=en_US:en' >> /home/builder/.bashrc

CMD ["/bin/bash"]
EOF

# -----------------------------
# 4. Build Docker image
# -----------------------------
log "Building Docker image: $IMAGE_NAME"
docker build -t "$IMAGE_NAME" "$DOCKER_CONTEXT"

# -----------------------------
# 5. Remove old container if exists
# -----------------------------
log "Removing old container if it exists: $CONTAINER_NAME"
if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    docker rm -f "$CONTAINER_NAME"
fi

# -----------------------------
# 6. Start container
# -----------------------------
log "Starting Jelina builder container"

docker run --privileged -it \
    --security-opt apparmor=unconfined \
    --name "$CONTAINER_NAME" \
    -v "$HOST_WORKSPACE:/home/builder/workspace" \
    -v "$HOST_DOWNLOADS:/home/builder/downloads" \
    -v "$HOST_OUTPUT:/home/builder/output" \
    -v "$HOST_ROOTFS:/home/builder/rootfs" \
    -v "$HOST_SCRIPTS:/home/builder/scripts" \
    -v /dev:/dev \
    "$IMAGE_NAME" \
    /bin/bash -lc '
        echo
        echo "Container started successfully."
        echo
        echo "Checking /bin/sh:"
        ls -l /bin/sh
        echo
        echo "Checking locale:"
        locale
        echo
        echo "Checking required HOSTTOOLS:"
        for t in chrpath diffstat gawk zstd pzstd unzstd newuidmap newgidmap; do
            printf "%-12s -> " "$t"
            command -v "$t" || true
        done
        echo
        echo "Checking user namespace:"
        if unshare -Ur true >/dev/null 2>&1; then
            echo "OK: user namespace is usable."
        else
            echo "WARNING: user namespace is NOT usable."
            echo "If BitBake fails, run this on the HOST:"
            echo "  sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0"
            echo "Then recreate the container with --security-opt apparmor=unconfined."
        fi
        echo
        echo "Next steps:"
        echo "  cd ~/workspace/JelinaBSP-rk-xiangsheng-v1.2.2-2026-02-10T16-22-09"
        echo "  source setup-environment"
        echo "  bitbake jelina-flasher-image"
        echo
        exec /bin/bash
    '
