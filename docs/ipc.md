# IPC Protocol

Minibox uses a single Unix socket (`ipc.sock`) for CLI-to-daemon requests. Each request is a JSON-encoded `CommandMessage`, and each response is a `CommandResult`.

## Message Shape

`CommandMessage`:

```json
{
  "name": "status",
  "payload": {},
  "meta": {}
}
```

`CommandResult`:

```json
{
  "status": "ok",
  "error": "",
  "data": {}
}
```

Notes:
- `status` is `"ok"` or `"error"`.
- When `status` is `"error"`, `error` contains a readable message.
- `data` is command-specific.

## Command List

Daemon handlers accept the following `name` values:

- `run`
  - Payload: `mode`, `route`, `api_port`, `mixed_port`
  - Effect: Start sing-box with generated config and persist runtime state.
- `check`
  - Payload: `config` (optional path)
  - Effect: Validate config build without running sing-box.
- `update`
  - Payload: none
  - Effect: Download rule data (geoip/geosite).
- `stop`
  - Payload: none
  - Effect: Stop the running sing-box instance.
- `status`
  - Payload: none
  - Data: `running`, `pid`, `proxy_mode`, `route_mode`, `listen_addr`, `api_port`, `mixed_port`
- `health`
  - Payload: none
  - Data: `running`, `pid`
- `reload`
  - Payload: none
  - Effect: Rebuild config from the current profile and reload sing-box.
- `mode`
  - Payload: `mode`
  - Data: `proxy_mode`
- `route`
  - Payload: `route`
  - Data: `route_mode`
- `node.list`
  - Payload: `api` (optional override)
  - Data: `proxies`
- `node.use`
  - Payload: `group`, `node`, `api` (optional override)
  - Data: `group`, `node`
- `log`
  - Payload: none
  - Data: `path` (log file path)

## Flow Overview

1. CLI serializes command arguments into `payload`.
2. CLI sends `CommandMessage` to the daemon socket.
3. Daemon handler performs the action and updates cached state.
4. CLI reads `CommandResult` and formats output.
