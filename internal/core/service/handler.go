package service

import (
	"context"

	"github.com/kyson/minibox/internal/adapter/logger"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
)

// Reload 处理配置重新加载请求
// CLI 已经更新了 raw.json，Daemon 只需要重新加载
func (s *instance) Reload(ctx context.Context, req *ipc.Request) *ipc.Response {
	logger.Info("Reloading configuration...")

	// 从 raw.json 重新加载配置
	if err := s.ReloadFromFile(ctx, env.Get().RawConfigFile); err != nil {
		return ipc.NewErrorResponse(req.ID, ipc.ErrCodeInternal, "failed to reload: "+err.Error())
	}

	logger.Info("Configuration reloaded successfully")
	resp, _ := ipc.NewResponse(req.ID, nil)
	return resp
}
