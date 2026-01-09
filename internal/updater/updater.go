package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kysonzou/sing-helm/internal/logger"
)

var (
	defaultHTTPClientFactory = func() *http.Client {
		return &http.Client{}
	}
	httpClientFactory = defaultHTTPClientFactory
)

type ProgressCallback func(current, total int64)

func Download(ctx context.Context, url, destDir, filename string, onProgress ProgressCallback) error {
	logger.Info("Starting download", "url", url)

	// 1. 创建Http请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// 2. 发送请求
	// 超时控制由 Context 统一管理，不在 Client 层设置 Timeout
	client := httpClientFactory()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	// 3. 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 4. 创建临时文件
	// 模式：先下载到 geoip.db.tmp，成功后再重命名为 geoip.db
	// 这样可以避免覆盖了旧文件结果新文件下载一半断了，导致旧文件也没了
	tmpPath := filepath.Join(destDir, filename+".tmp")
	destPath := filepath.Join(destDir, filename)

	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file error: %w", err)
	}
	defer file.Close()

	// 5. 准备写入 (带进度监控)
	contentLength := resp.ContentLength

	// 定义一个 wrapper 来统计写入字节数
	var reader io.Reader = resp.Body
	if onProgress != nil {
		reader = &progressReader{
			Reader:   resp.Body,
			Total:    contentLength,
			Callback: onProgress,
		}
	}

	// 6. 数据拷贝 (核心 IO 操作)
	// io.Copy 会一直阻塞直到下载完成或报错
	if _, err := io.Copy(file, reader); err != nil {
		file.Close()       // 必须先关闭文件句柄
		os.Remove(tmpPath) // 删除下载了一半的垃圾文件
		return fmt.Errorf("download failed: %w", err)
	}
	file.Close() // 写入完成，关闭文件

	// 7. 原子重命名 (Atomic Rename)
	// 在 Linux/macOS 上是原子的，Windows 上如果文件存在可能会报错，先尝试删除
	if err := os.Rename(tmpPath, destPath); err != nil {
		// 针对 Windows 的兼容处理：如果重命名失败，可能是目标文件已存在且被占用
		// 这里简单处理：先 Remove 再 Rename
		_ = os.Remove(destPath)
		if err := os.Rename(tmpPath, destPath); err != nil {
			return fmt.Errorf("failed to save file: %w", err)
		}
	}

	logger.Info("Download saved", "path", destPath)
	return nil
}

// SetHTTPClientFactory 用于测试，用自定义的 HTTP 客户端替换默认实现。
func SetHTTPClientFactory(factory func() *http.Client) {
	if factory == nil {
		return
	}
	httpClientFactory = factory
}

// ResetHTTPClientFactory 恢复默认的 HTTP 客户端行为。
func ResetHTTPClientFactory() {
	httpClientFactory = defaultHTTPClientFactory
}

type progressReader struct {
	io.Reader
	Total    int64
	Current  int64
	Callback ProgressCallback
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Callback != nil {
		pr.Callback(pr.Current, pr.Total)
	}
	return n, err
}
