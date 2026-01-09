package updater_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyson-dev/sing-helm/internal/updater"
	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUpdater_DownLoad(t *testing.T) {
	// 1. 构造一个 Handler 模拟下载内容
	mockContent := "This is a fake geoip database content"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "37")
		_, _ = w.Write([]byte(mockContent))
	})

	trans := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		resp := recorder.Result()
		resp.Request = req
		return resp, nil
	})

	client := &http.Client{Transport: trans}
	updater.SetHTTPClientFactory(func() *http.Client {
		return client
	})
	defer updater.ResetHTTPClientFactory()

	tmpDir := t.TempDir()
	downloadedBytes := int64(0)
	err := updater.Download(context.Background(), "http://fake.test", tmpDir, "test.db", func(current, total int64) {
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
