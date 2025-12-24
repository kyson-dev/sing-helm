package cli_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/kyson/minibox/internal/adapter/cli"
	"github.com/kyson/minibox/internal/env"
	"github.com/stretchr/testify/assert"
)

func TestCLI_VersionCommand(t *testing.T) {
	cmd := cli.NewRootCommand()
	cmd.SetOut(bytes.NewBufferString(""))
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

func TestCLI_CheckCommand(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "valid config",
			configPath: createTempConfig(t, `{"inbounds": [{"type": "tun"}], "outbounds": [{"type": "direct"}]}`),
			wantErr:    false,
		},
		{
			name:       "invalid json format",
			configPath: createTempConfig(t, `{"invalid": json}`),
			wantErr:    true,
		},
		{
			name:       "non-existent file",
			configPath: "non_existent.json",
			wantErr:    true,
		},
		// 移除 empty config 测试，因为现在空配置会自动生成默认的 direct outbound 和 mixed inbound
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		name       string
		configPath string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "non-existent config file",
			configPath: "non_existent_config.json",
			wantErr:    true,
			errMsg:     "failed to load profile file",
		},
		{
			name:       "invalid json config",
			configPath: createTempConfig(t, `{"invalid": json}`),
			wantErr:    true,
			errMsg:     "failed to load profile file",
		},
		{
			name:       "empty config - should start successfully with defaults",
			configPath: createTempConfig(t, `{}`),
			wantErr:    false, // 现在的代码会自动补全默认 outbound，所以应该能启动成功
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置环境，确保每次都可以重新初始化路径
			env.Reset()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// 创建临时的 home 目录
			tmpHome := t.TempDir()

			// 如果提供了 configPath (临时文件路径)，将其复制/重命名为 profile.json 放到 tmpHome 下
			// 如果是 "non_existent"，则不创建
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
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
