# Minibox system service install

This project runs a system-level daemon with user-level CLI control.

## Autostart (recommended)

Enable autostart:

```sh
sudo minibox autostart on
```

Disable autostart:

```sh
sudo minibox autostart off
```

## Manual install (optional)

If you prefer to manage service files yourself, use the scripts below.

### Linux (systemd)

```sh
sudo scripts/systemd/install-systemd.sh
```

Enable/start:

```sh
sudo scripts/systemd/enable-systemd.sh
```

Disable/stop:

```sh
sudo scripts/systemd/disable-systemd.sh
```

Grant a user access to the socket:

```sh
sudo usermod -aG minibox <username>
```

Uninstall:

```sh
sudo scripts/systemd/uninstall-systemd.sh
```

Purge logs:

```sh
sudo scripts/systemd/uninstall-systemd.sh --purge
```

### macOS (launchd)

Install service files (does not enable/start):

```sh
sudo scripts/launchd/install-launchd.sh
```

Enable/start:

```sh
sudo scripts/launchd/enable-launchd.sh
```

You can also initialize runtime dirs only (no service enable):

```sh
sudo scripts/init-runtime.sh
```

Disable/stop:

```sh
sudo scripts/launchd/disable-launchd.sh
```

Grant a user access to the socket:

```sh
sudo dscl . -append /Groups/minibox GroupMembership <username>
```

Uninstall:

```sh
sudo scripts/launchd/uninstall-launchd.sh
```

Purge logs:

```sh
sudo scripts/launchd/uninstall-launchd.sh --purge
```

## Log location

By default logs go to:

- Linux/macOS: `/var/log/minibox/minibox.log`

If the log directory is not writable, Minibox falls back to the runtime directory.
