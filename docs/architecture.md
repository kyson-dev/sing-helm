# 架构概览

本文面向新同学快速理解 minibox 的整体结构与运行方式，基于当前的 IPC + daemon 架构。

## 总览

minibox 是一个 Go CLI + daemon 的工程：
- CLI 负责命令解析与参数序列化。
- daemon 负责状态管理、配置构建、sing-box 启动与控制。
- 通过 Unix Socket 的 IPC 协议交换 `CommandMessage`/`CommandResult`。

## 目录结构与职责

- `cmd/minibox`
  - CLI 入口，构建 `minibox` 二进制。
- `internal/adapter/cli`
  - CLI 命令与输出；仅做参数校验与 IPC 调度。
- `internal/core/daemon`
  - daemon 命令处理中心，负责状态管理与 handler 路由。
- `internal/core/service`
  - sing-box 生命周期封装（Start/Reload/Run）。
- `internal/core/runtime`
  - 运行期配置构建管线（profile -> raw.json）。
- `internal/core/config`
  - 配置模型、模块化构建器与运行状态结构。
- `internal/ipc`
  - IPC 协议与 Unix socket sender/server。
- `internal/env`
  - 路径、锁、环境初始化与注册。
- `internal/ui/monitor`
  - TUI 监控界面。
- `docs/`
  - 文档与设计说明。

## 关键数据流

### 1) CLI 命令

1. 用户执行 CLI 子命令（如 `mode`, `node`, `status`）。
2. CLI 构造 `CommandMessage` 并发送到 daemon socket。
3. daemon 路由到 handler，执行业务逻辑并返回 `CommandResult`。
4. CLI 读取结果并渲染输出。

### 2) 启动与运行

1. CLI `start`/`daemon` 触发 daemon 进程。
2. daemon `run` handler:
   - 解析参数为 `RunOptions`。
   - 调用 `runtime.BuildConfig` 生成 raw.json。
   - 启动 sing-box 服务并维护 runtime state。
3. CLI/monitor 通过 IPC 查询状态。

### 3) 配置构建

- `runtime.BuildConfig(profilePath, rawPath, runops)`：
  - 加载 profile.json
  - 应用模块化配置（`DefaultModules`）
  - 写入 raw.json

## IPC 协议

- IPC 使用 `CommandMessage`/`CommandResult` 结构体。
- 详细命令列表见 `docs/ipc.md`。

## 运行时状态

- daemon 在内存中持有 runtime state，同时持久化到 state.json。
- CLI/monitor 不再直接读取 state 文件，统一通过 IPC 查询状态。

## 设计原则

- CLI 只负责参数 -> IPC，不直接做核心业务。
- daemon 是唯一的业务与状态入口。
- 配置构建与运行期控制解耦，便于测试与复用。
- 内部包遵守 `internal/` 边界，避免跨层调用。
