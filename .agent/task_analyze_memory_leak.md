# Task: Analyze Memory Leak

## Status
- [x] Analyze project structure and core components
- [x] Review specific modules (daemon, service, ipc, cli, tui, config, updater)
- [x] Identify potential memory leak sources (sing-box core vs sing-helm code)
- [x] Verify upstream sing-box issues
- [x] Report findings to user (Confirmed Hysteria2 memory leak bug in sing-box)

## Findings
- The memory leak (~900MB growth) is primarily caused by an upstream bug in `sing-box` regarding the **Hysteria2 protocol** (Issue #3421).
- **Remote Rule Sets**: While they consume memory, they are not the cause of the *growth*.
- **Go Code Analysis**: `sing-helm`'s internal Go code (daemon, TUI, etc.) follows good practices and does not show obvious leak patterns.

## Resolution
- User decided to reduce usage of Hysteria2 protocol to mitigate the issue.
- No code changes required in `sing-helm` at this moment.
- Recommended to wait for `sing-box` upstream fix before upgrading.
