#!/bin/bash
set -e

# SingHelm Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/kysonzou/sing-helm/main/scripts/install.sh | bash

REPO="kysonzou/sing-helm"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="sing-helm"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        *)
            error "Unsupported OS: $os"
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest release version from GitHub
get_latest_version() {
    info "Fetching latest release..."
    VERSION=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version"
    fi
    
    info "Latest version: $VERSION"
}

# Download binary
download_binary() {
    local binary_name="${BINARY_NAME}-${PLATFORM}"
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${binary_name}"
    local tmp_file="/tmp/${binary_name}"

    info "Downloading from: $download_url"
    
    if ! curl -sSL -o "$tmp_file" "$download_url"; then
        error "Failed to download binary"
    fi

    chmod +x "$tmp_file"
    DOWNLOADED_BINARY="$tmp_file"
    info "Downloaded to: $tmp_file"
}

# Install binary
install_binary() {
    local target="${INSTALL_DIR}/${BINARY_NAME}"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        info "Installing to $target"
        mv "$DOWNLOADED_BINARY" "$target"
    else
        info "Installing to $target (requires sudo)"
        sudo mv "$DOWNLOADED_BINARY" "$target"
    fi

    info "✅ Installation complete!"
}

# Print post-install instructions
print_instructions() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  SingHelm installed successfully!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Quick Start:"
    echo "  1. Check version:"
    echo "     $ sing-helm version"
    echo ""
    echo "  2. View help:"
    echo "     $ sing-helm --help"
    echo ""
    echo "  3. Start service:"
    echo "     $ sudo sing-helm run"
    echo ""
    echo "  4. Enable autostart (macOS):"
    echo "     $ sudo sing-helm autostart on"
    echo ""
    echo "For more information, visit:"
    echo "  https://github.com/${REPO}"
    echo ""
}

# Main installation flow
main() {
    info "Starting SingHelm installation..."
    
    detect_platform
    get_latest_version
    download_binary
    install_binary
    print_instructions
}

main

