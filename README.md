# Sing-Helm

> **A CLI-first, lightweight manager for [sing-box](https://github.com/sagernet/sing-box).**  
> Helm your network traffic with ease.

[![License](https://img.shields.io/github/license/kyson-dev/sing-helm)](LICENSE)
[![Release](https://img.shields.io/github/v/release/kyson-dev/sing-helm)](https://github.com/kyson-dev/sing-helm/releases)
[![Build Status](https://github.com/kyson-dev/sing-helm/actions/workflows/ci.yaml/badge.svg)](https://github.com/kyson-dev/sing-helm/actions)

**Sing-Helm** is a powerful command-line interface designed to simplify the deployment and management of sing-box on Linux and macOS. It handles configuration generation, subscription management, and service lifecycle, making it the perfect companion for advanced proxy users.

---

## ✨ Features

*   **⚡️ Zero Config Start**: Run immediately with intelligent defaults.
*   **📦 Subscription Management**: parse and convert subscriptions to sing-box config automatically.
*   **🔄 Automatic Updates**: Built-in Homebrew tap integration for seamless upgrades.
*   **🖥️ Daemon Management**: Built-in service manager (Systemd / Launchd) for autostart.
*   **🛠️ Hot Reload**: Apply configuration changes without dropping connections.
*   **🍎 macOS & 🐧 Linux**: First-class support for both platforms (AMD64 & ARM64).

---

## 🚀 Installation

### Option 1: Homebrew (Recommended for macOS)

The easiest way to install and keep updated.

```bash
brew tap kyson-dev/sing-helm
brew install sing-helm
```

### Option 2: Shell Script (Recommended for Linux)

Install with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/kyson-dev/sing-helm/main/scripts/install.sh | bash
```

### Option 3: Manual

Download the latest binary from the [Releases](https://github.com/kyson-dev/sing-helm/releases) page and add it to your PATH.

---

## 📖 Quick Start

### 1. Start the Service

Sing-Helm runs as a daemon to manage the proxy core.

```bash
# Start in foreground (for testing)
sing-helm run

# Start as a system service (background)
sudo sing-helm start
```

### 2. Configure Subscription

Add your subscription URL (supports standard subscription formats):

*(Feature coming soon: `sing-helm sub add <url>`)*

Currently, put your subscription info into `~/.sing-helm/profile.json`.

### 3. Check Status

```bash
# Check if the service is running
sudo sing-helm status

# View logs
sing-helm log
```

---

## 🛠️ Commands

| Command | Description |
| :--- | :--- |
| `sing-helm run` | Run the daemon in the foreground (useful for debug) |
| `sing-helm start` | Start the system service (Ubuntu/Debian/macOS) |
| `sing-helm stop` | Stop the system service |
| `sing-helm restart` | Restart the system service |
| `sing-helm status` | Check service status |
| `sing-helm log` | Follow real-time logs |
| `sing-helm autostart on`| Enable service to start on boot |
| `sing-helm version` | Show version information |

---

## 📂 Configuration

The default configuration directory is `~/.sing-helm`.

*   **profile.json**: Your user settings (subscriptions, rules).
*   **config.json**: The generated sing-box configuration (do not edit manually).
*   **sing-helm.log**: Runtime logs.

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1.  Fork the project
2.  Create your feature branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
