package updater_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyson/minibox/internal/core/updater"
	"github.com/stretchr/testify/assert"
)

func TestUpdater_DownLoad(t *testing.T) {
	// 1. 启动一个 Mock Server
	// 当我们访问这个 server 时，它返回一段假数据
	mockContent :=  "This is a fake geoip database content"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "37")
		_, _ = w.Write([]byte(mockContent))
	}))
	defer ts.Close()

	tmpDir := t.TempDir() // 文件夹
	// 执行下载
	downloadedBytes := int64(0)
	err := updater.Download(context.Background(), ts.URL, tmpDir, "test.db", func(current, total int64) {
		downloadedBytes = current
	})

	// 4. 断言
	assert.NoError(t, err)

	// 验证文件存在
	destPath := filepath.Join(tmpDir, "test.db")
	assert.FileExists(t, destPath)

	// 验证内容正确
	content, _ := os.ReadFile(destPath)
	assert.Equal(t, mockContent, string(content))

	// 验证回调是否执行
	assert.Equal(t, int64(len(mockContent)), downloadedBytes)
}