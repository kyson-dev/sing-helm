package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kyson/minibox/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Success(t *testing.T) {
	// 1. 定位测试文件
	// 注意：go test 运行时，工作目录是当前包的目录
	// 所以我们要往上找几层去找到 testdata
	// "../../../testdata/config.json"
	path := filepath.Join("..", "..", "..", "testdata", "config.test.json")

	// 2. 加载
	opts, err := config.Load(path)

	// 3. 断言没有错误
	// require.NoError 如果失败会立即终止测试，适合检查 err
	require.NoError(t, err)
	require.NotNil(t, opts)

	// 4. 断言配置正确
	// 检查我们在 testdata/config.json 里写的值
	assert.Equal(t, "info", opts.Log.Level)
	assert.Equal(t, 1, len(opts.Inbounds))
	assert.Equal(t, "mixed-in", opts.Inbounds[0].Tag)
	assert.Equal(t, "127.0.0.1:19090", opts.Experimental.ClashAPI.ExternalController)
}

func TestLoad_FileNotFound(t *testing.T) {
	// 1. 定位测试文件
	path := filepath.Join("not-exist.json")

	// 2. 加载
	opts, err := config.Load(path)

	// 3. 断言错误
	require.Error(t, err)
	require.Nil(t, opts)
}

func TestLoad_InvalidConfig(t *testing.T) {
	// 1. 临时目录
	tmp := t.TempDir()

	// 2. 创建临时文件
	badConfigPath := filepath.Join(tmp, "invalid-config.json")
	os.WriteFile(badConfigPath, []byte("invalid json"), 0644)

	// 2. 加载
	opts, err := config.Load(badConfigPath)

	// 3. 断言错误
	require.Error(t, err)
	require.Nil(t, opts)
}
