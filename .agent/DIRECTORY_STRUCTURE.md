# sing-helm 目录规划

## 设计原则

1. **Runtime 目录**: 存放运行时临时数据（socket, lock, state, cache）
   - 需要持久化（重启不丢失）
   - 需要快速访问

2. **Log 目录**: 存放日志文件
   - 系统标准位置
   - 需要持久化

3. **Home 目录**: 存放用户配置
   - 用户级别的配置文件
   - 订阅配置等

## 各平台目录规划

### macOS (darwin)

| 类型 | 路径 | 说明 |
|------|------|------|
| Runtime | `/usr/local/var/run/sing-helm` | 持久化，避免 /var/run 重启清空 |
| Log | `/var/log/sing-helm` | 系统标准日志位置 |
| Home | `~/.sing-helm` | 用户配置目录 |

**Runtime 目录内容**:
- `ipc.sock` - IPC socket
- `sing-helm.lock` - 进程锁
- `state.json` - 运行状态
- `runtime.json` - 运行时元数据
- `raw.json` - 生成的完整配置
- `cache.db` - sing-box 缓存
- `assets/` - geoip.db, geosite.db

**Log 目录内容**:
- `sing-helm.log` - 应用日志（主要）
- `stdout.log` - launchd 捕获的标准输出（仅自动启动时）
- `stderr.log` - launchd 捕获的错误输出（仅自动启动时）

**Home 目录内容**:
- `profile.json` - 用户配置
- `subscriptions/` - 订阅配置
- `subscriptions/cache/` - 订阅缓存

### Linux

| 类型 | 路径 | 说明 |
|------|------|------|
| Runtime | `/run/sing-helm` 或 `/var/run/sing-helm` | 优先使用 /run |
| Log | `/var/log/sing-helm` | 系统标准日志位置 |
| Home | `~/.sing-helm` | 用户配置目录 |

**注意**: Linux 上 systemd 会自动管理日志到 journald，可用 `journalctl -u sing-helm` 查看。

### Windows

| 类型 | 路径 | 说明 |
|------|------|------|
| Runtime | `%ProgramData%\sing-helm` | 或 Temp 目录 |
| Log | `%ProgramData%\sing-helm\logs` | 或 Temp 目录 |
| Home | `~/.sing-helm` | 用户配置目录 |

## 代码实现位置

- **Runtime 目录解析**: `internal/env/runtime.go` → `ResolveRuntimeDir()`
- **Log 目录解析**: `internal/logger/log.go` → `ResolveLogDir(runtimeDir)`
- **路径组装**: `internal/env/paths.go` → `GetPath(home, runtimeDir, logDir)`

## 环境变量覆盖

可通过环境变量覆盖默认路径：
- `SINGHELM_RUNTIME_DIR` - 覆盖 runtime 目录

## 权限要求

- **Runtime 目录**: 需要写权限
- **Log 目录**: 需要写权限（如果没有权限，会降级到 runtime 目录）
- **Home 目录**: 用户权限即可

## 自动启动配置

### macOS (launchd)
- Plist 路径: `/Library/LaunchDaemons/com.kyson.sing-helm.plist`
- 二进制路径: 动态获取（`os.Executable()`）
- 日志路径: 动态获取（`logger.ResolveLogDir()`）

### Linux (systemd)
- Unit 路径: `/etc/systemd/system/sing-helm.service`
- 二进制路径: 动态获取（`os.Executable()`）
- 日志: systemd journald

## 一致性检查

✅ macOS Runtime: `/usr/local/var/run/sing-helm` (持久化)
✅ macOS Log: `/var/log/sing-helm` (系统标准)
✅ Linux Runtime: `/run/sing-helm` (标准)
✅ Linux Log: `/var/log/sing-helm` (系统标准)
✅ 二进制路径: 动态获取，不硬编码
✅ 日志路径: 动态获取，不硬编码
