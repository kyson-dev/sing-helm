package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyson/minibox/internal/adapter/cli"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	runtimeDir := filepath.Join(os.TempDir(), "minibox-runtime-test")
	os.RemoveAll(runtimeDir)
	os.MkdirAll(runtimeDir, 0755)
	env.SetRuntimeDir(runtimeDir)
	defer env.ResetRuntimeDir()
	defer os.RemoveAll(runtimeDir)

	os.Setenv("MINIBOX_TEST_SKIP_SERVICE", "1")
	os.Setenv("MINIBOX_TEST_MIXED_PORT", "10808")
	os.Setenv("MINIBOX_TEST_API_PORT", "18080")
	cli.SetCommandSenderFactory(func() ipc.CommandSender {
		return &ipc.FakeSender{Response: ipc.CommandResult{Status: "ok"}}
	})
	defer cli.ResetCommandSenderFactory()

	os.Exit(m.Run())
}

func TestCLI_VersionCommand(t *testing.T) {
	cmd := cli.NewRootCommand()
	cmd.SetOut(bytes.NewBufferString(""))
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

func TestCLI_CheckCommand(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		wantErr       bool
		forceFallback bool
	}{
		{
			name:       "valid config",
			configPath: createTempConfig(t, `{"inbounds": [{"type": "tun"}], "outbounds": [{"type": "direct"}]}`),
			wantErr:    false,
		},
		{
			name:          "invalid json format",
			configPath:    createTempConfig(t, `{"invalid": json}`),
			wantErr:       true,
			forceFallback: true,
		},
		{
			name:          "non-existent file",
			configPath:    "non_existent.json",
			wantErr:       true,
			forceFallback: true,
		},
		// 移除 empty config 测试，因为现在空配置会自动生成默认的 direct outbound 和 mixed inbound
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.forceFallback {
				cli.SetCommandSenderFactory(func() ipc.CommandSender {
					return &ipc.FakeSender{Err: fmt.Errorf("ipc: connect failed: fallback")}
				})
				t.Cleanup(cli.ResetCommandSenderFactory)
			}

			root := cli.NewRootCommand()
			root.SetArgs([]string{"check", "--config", tt.configPath})
			err := root.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCLI_RunCommand 测试 run 命令的错误处理
// 注意：完整的集成测试需要真实的 sing-box 环境，这里只测试配置加载
func TestCLI_RunCommand(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		wantErr       bool
		errMsg        string
		forceFallback bool
	}{
		{
			name:          "non-existent config file - should create default",
			configPath:    "non_existent_config.json",
			wantErr:       false,
			errMsg:        "",
			forceFallback: false,
		},
		{
			name:          "invalid json config - should create default",
			configPath:    createTempConfig(t, `{"invalid": json}`),
			wantErr:       false,
			errMsg:        "",
			forceFallback: false,
		},
		{
			name:          "empty config - should start successfully with defaults",
			configPath:    createTempConfig(t, `{}`),
			wantErr:       false, // 现在的代码会自动补全默认 outbound，所以应该能启动成功
			forceFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置环境，确保每次都可以重新初始化路径
			env.ResetForTest()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// 创建临时的 home 目录 (使用 /tmp 以避免路径过长导致 unix socket bind 失败)
			tmpHome := filepath.Join("/tmp", fmt.Sprintf("minibox-test-%d-%d", time.Now().UnixNano(), os.Getpid()))
			os.MkdirAll(tmpHome, 0755)
			defer os.RemoveAll(tmpHome)
			env.SetRuntimeDir(tmpHome)

			// 如果提供了 configPath (临时文件路径)，将其复制/重命名为 profile.json 放到 tmpHome 下
			// 如果是 "non_existent"，则不创建
			if tt.forceFallback {
				cli.SetCommandSenderFactory(func() ipc.CommandSender {
					return &ipc.FakeSender{Err: fmt.Errorf("ipc: connect failed: fallback")}
				})
				t.Cleanup(cli.ResetCommandSenderFactory)
			}

			if tt.configPath != "" && tt.configPath != "non_existent_config.json" {
				content, err := os.ReadFile(tt.configPath)
				assert.NoError(t, err)
				err = os.WriteFile(tmpHome+"/profile.json", content, 0644)
				assert.NoError(t, err)
			}

			// 如果是测试 "non_existent"，我们什么都不放，run 命令会在 tmpHome 下找 profile.json 找不到

			root := cli.NewRootCommand()
			// 使用 --home 指定工作目录
			root.SetArgs([]string{"run", "--home", tmpHome})
			root.SetContext(ctx)

			err := root.Execute()

			if tt.wantErr {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCLI_ModeCommandStatus(t *testing.T) {
	var output bytes.Buffer
	resp := ipc.CommandResult{
		Status: "ok",
		Data:   map[string]any{"proxy_mode": "tun"},
	}
	cli.SetCommandSenderFactory(func() ipc.CommandSender {
		return &ipc.FakeSender{Response: resp}
	})
	t.Cleanup(cli.ResetCommandSenderFactory)

	cmd := cli.NewRootCommand()
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"mode"})

	assert.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "Current proxy mode: tun")
}

func TestCLI_ModeCommandSwitch(t *testing.T) {
	var output bytes.Buffer
	cli.SetCommandSenderFactory(func() ipc.CommandSender {
		return &ipc.FakeSender{Response: ipc.CommandResult{Status: "ok"}}
	})
	t.Cleanup(cli.ResetCommandSenderFactory)

	cmd := cli.NewRootCommand()
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"mode", "default"})

	assert.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "Proxy mode switched to: default")
}

func TestCLI_RouteCommandStatus(t *testing.T) {
	var output bytes.Buffer
	resp := ipc.CommandResult{
		Status: "ok",
		Data:   map[string]any{"route_mode": "global"},
	}
	cli.SetCommandSenderFactory(func() ipc.CommandSender {
		return &ipc.FakeSender{Response: resp}
	})
	t.Cleanup(cli.ResetCommandSenderFactory)

	cmd := cli.NewRootCommand()
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"route"})

	assert.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "Current route mode: global")
}

func TestCLI_RouteCommandSwitch(t *testing.T) {
	var output bytes.Buffer
	cli.SetCommandSenderFactory(func() ipc.CommandSender {
		return &ipc.FakeSender{Response: ipc.CommandResult{Status: "ok"}}
	})
	t.Cleanup(cli.ResetCommandSenderFactory)

	cmd := cli.NewRootCommand()
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"route", "direct"})

	assert.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "Route mode switched to: direct")
}

// createTempConfig 创建临时配置文件用于测试
func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_config_*.json")
	assert.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()
	return tmpFile.Name()
}
