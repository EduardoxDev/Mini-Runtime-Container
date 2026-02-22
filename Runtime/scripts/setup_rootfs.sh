#!/bin/bash
# setup_rootfs.sh — Download and extract Alpine minirootfs
# Usage: ./scripts/setup_rootfs.sh [target_dir]

set -euo pipefail

ALPINE_VERSION="3.19"
ALPINE_ARCH="x86_64"
TARGET_DIR="${1:-rootfs}"
ROOTFS_URL="https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/releases/${ALPINE_ARCH}/alpine-minirootfs-${ALPINE_VERSION}.0-${ALPINE_ARCH}.tar.gz"

echo "==> GoContainer Rootfs Setup"
echo "    Alpine version: ${ALPINE_VERSION}"
echo "    Architecture:   ${ALPINE_ARCH}"
echo "    Target:         ${TARGET_DIR}"
echo ""

# Check if already extracted
if [ -f "${TARGET_DIR}/.extracted" ]; then
    echo "==> Rootfs already extracted at ${TARGET_DIR}"
    exit 0
fi

# Create target directory
mkdir -p "${TARGET_DIR}"

# Download and extract
echo "==> Downloading Alpine minirootfs from ${ROOTFS_URL}..."
if command -v curl &> /dev/null; then
    curl -fSL "${ROOTFS_URL}" | tar xz -C "${TARGET_DIR}"
elif command -v wget &> /dev/null; then
    wget -qO- "${ROOTFS_URL}" | tar xz -C "${TARGET_DIR}"
else
    echo "ERROR: Neither curl nor wget found. Please install one of them."
    exit 1
fi

# Mark as extracted
touch "${TARGET_DIR}/.extracted"

echo "==> Rootfs extracted successfully to ${TARGET_DIR}"
echo "==> Contents:"
ls -la "${TARGET_DIR}/"
