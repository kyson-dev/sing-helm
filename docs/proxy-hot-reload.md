# 系统代理热切换功能

## 功能说明

通过 Go API 实现系统代理的热切换，无需重启 sing-box。

## 使用方法

### 1. 启动 sing-box (默认模式，不开启系统代理)
```bash
./bin/minibox run -c testdata/config.json
```

### 2. 在另一个终端中，动态开启系统代理
```bash
./bin/minibox proxy on
```

### 3. 关闭系统代理
```bash
./bin/minibox proxy off
```

## 实现原理

1. **热更新机制**:
   - 使用 `box.Inbound().Remove("mixed-in")` 删除旧的 inbound
   - 使用 `box.Inbound().Create(...)` 创建新的 inbound，设置 `set_system_proxy: true/false`
   - 无需重启整个 sing-box 进程

2. **跨命令通信**:
   - 使用全局实例 `service.GlobalInstance`
   - `run` 命令设置全局实例
   - `proxy` 命令通过全局实例调用 `SetSystemProxy()`

3. **状态管理**:
   - 从 `state.json` 读取端口信息
   - 确保使用相同的监听地址和端口

## 限制

- ⚠️ **仅在同一进程内有效**: `proxy on/off` 必须在 `minibox run` 运行时调用
- ⚠️ **全局变量方案**: 当前使用全局变量，生产环境应使用 IPC (如 Unix Socket)
- ✅ **不影响现有连接**: 热切换不会断开已建立的连接

## 未来改进

1. **IPC 通信**: 使用 Unix Socket 或 gRPC 实现进程间通信
2. **Daemon 模式**: 将 `run` 改为后台守护进程
3. **更多热更新**: 支持 TUN 模式切换、节点添加等

## 测试

```bash
# Terminal 1: 启动 sing-box
./bin/minibox run -c testdata/config.json

# Terminal 2: 测试热切换
./bin/minibox proxy on   # 开启系统代理
./bin/minibox proxy off  # 关闭系统代理
./bin/minibox proxy on   # 再次开启
```

查看系统代理设置是否实时变化！
