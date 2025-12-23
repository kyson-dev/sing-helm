package service_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/kyson/minibox/internal/core/service"
	"github.com/sagernet/sing-box/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestService_NewInstance 测试创建新实例
func TestService_NewInstance(t *testing.T) {
	instance := service.NewInstance()
	assert.NotNil(t, instance)
}

// TestService_CloseNilInstance 测试关闭未启动的实例
func TestService_CloseNilInstance(t *testing.T) {
	instance := service.NewInstance()
	ctx := context.Background()

	// 关闭未启动的实例应该不报错
	err := instance.Close(ctx)
	assert.NoError(t, err)
}

// TestService_MultipleCloseNilInstance 测试多次关闭未启动的实例
func TestService_MultipleCloseNilInstance(t *testing.T) {
	instance := service.NewInstance()
	ctx := context.Background()

	// 多次关闭未启动的实例都不应该报错
	err := instance.Close(ctx)
	assert.NoError(t, err)

	err = instance.Close(ctx)
	assert.NoError(t, err)
}

// TestService_StartWithNilOptions 测试使用 nil 配置启动
func TestService_StartWithNilOptions(t *testing.T) {
	instance := service.NewInstance()
	ctx := context.Background()

	// nil options 应该返回错误
	err := instance.Start(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "options cannot be nil")
}

// TestService_StartWithEmptyOptions 测试使用空配置启动
func TestService_StartWithEmptyOptions(t *testing.T) {
	instance := service.NewInstance()
	ctx := context.Background()

	// 空配置在 sing-box v1.10+ 中可能会启动成功
	err := instance.Start(ctx, &option.Options{})

	// 如果启动成功，需要清理
	if err == nil {
		t.Log("Empty config started successfully (sing-box version allows this)")
		defer instance.Close(ctx)
	} else {
		// 如果失败，验证错误信息
		assert.Contains(t, err.Error(), "failed to create box instance")
	}
}

// 获取一个空闲的本地端口
func getFreePort() int {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// createTestOptions 创建测试用的配置
func createTestOptions(t *testing.T, port int) *option.Options {
	t.Helper()

	rawConfig := map[string]any{
		"inbounds": []map[string]any{
			{
				"type":        "mixed",
				"tag":         "test-in",
				"listen":      "127.0.0.1",
				"listen_port": port,
			},
		},
		"outbounds": []map[string]any{
			{
				"type": "direct",
				"tag":  "direct-out",
			},
		},
	}

	// 将 map 转回 JSON bytes
	configBytes, err := json.Marshal(rawConfig)
	require.NoError(t, err)

	// 再解析成 sing-box 认识的 Options 结构体
	var opts option.Options
	err = json.Unmarshal(configBytes, &opts)
	require.NoError(t, err, "Failed to unmarshal test config")

	return &opts
}

func TestInstance_Lifecycle(t *testing.T) {
	// 1. 准备配置
	port := getFreePort()
	opts := createTestOptions(t, port)

	// 2. 启动服务
	svc := service.NewInstance()
	ctx := context.Background()

	err := svc.Start(ctx, opts)
	require.NoError(t, err, "Service should start without error")

	// 3. 验证服务状态
	time.Sleep(100 * time.Millisecond)
	assert.True(t, svc.IsRunning(), "Service should be running")

	// 4. 关闭服务
	err = svc.Close(ctx)
	assert.NoError(t, err, "Service should close without error")
	assert.False(t, svc.IsRunning(), "Service should not be running after close")
}

// TestInstance_ContextCancellation 测试通过 context 取消来关闭服务
func TestInstance_ContextCancellation(t *testing.T) {
	port := getFreePort()
	opts := createTestOptions(t, port)

	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动服务
	svc := service.NewInstance()
	err := svc.Start(ctx, opts)
	require.NoError(t, err, "Service should start without error")

	// 给服务一些启动时间
	time.Sleep(100 * time.Millisecond)
	assert.True(t, svc.IsRunning(), "Service should be running")
	// 通过取消 context 来关闭服务
	cancel()

	// 等待 goroutine 处理取消信号并清理
	time.Sleep(500 * time.Millisecond)

	// 验证服务已关闭
	assert.False(t, svc.IsRunning(), "Service should be closed")
}

// TestInstance_MultipleClose 测试重复关闭
func TestInstance_MultipleClose(t *testing.T) {
	port := getFreePort()
	opts := createTestOptions(t, port)

	svc := service.NewInstance()
	ctx := context.Background()

	// 启动服务
	err := svc.Start(ctx, opts)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// 第一次关闭
	err = svc.Close(ctx)
	assert.NoError(t, err)

	// 第二次关闭应该也不报错
	err = svc.Close(ctx)
	assert.NoError(t, err)
}
