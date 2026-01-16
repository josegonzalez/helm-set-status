#!/bin/bash

set -e

# Helm plugin install script
# This script is called by `helm plugin install`

cd "$HELM_PLUGIN_DIR"

version="$(grep "version:" plugin.yaml | cut -d '"' -f 2)"
echo "Installing helm-set-status v${version}..."

# Determine OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Determine file extension
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    EXT="zip"
fi

# Download URL
DOWNLOAD_URL="https://github.com/josegonzalez/helm-set-status/releases/download/v${version}/helm-set-status_${version}_${OS}_${ARCH}.${EXT}"

echo "Downloading from: $DOWNLOAD_URL"

# Create bin directory
mkdir -p bin

# Download and extract
if [ "$EXT" = "zip" ]; then
    curl -sSL "$DOWNLOAD_URL" -o /tmp/helm-set-status.zip
    unzip -o /tmp/helm-set-status.zip -d bin/
    rm /tmp/helm-set-status.zip
else
    curl -sSL "$DOWNLOAD_URL" | tar -xz -C bin/
fi

# Make binary executable
chmod +x bin/helm-set-status*

echo "helm-set-status v${version} installed successfully!"
