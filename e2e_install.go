//go:build e2e

package main

import (
	"context"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"agentre/internal/model/entity/agent_backend_entity"
	"agentre/internal/pkg/agentruntime"
	fakert "agentre/internal/pkg/agentruntime/runtimes/fake"
	"agentre/internal/repository/agent_backend_repo"
	"agentre/internal/repository/agent_repo"
	"agentre/internal/service/agent_backend_svc"
	"agentre/internal/service/agent_svc"
)

// installE2EFakes 仅在 `-tags e2e` 构建中编译:
//  1. 用确定性 fake 覆盖 claudecode runtime(无子进程/无登录);
//  2. seed 一个本地 claudecode backend 并挂到默认 CEO agent,
//     让前端"建会话→发消息→看回复"无需真实 CLI 即可跑通。
//
// 失败只记日志不 panic:e2e 环境异常应让 Playwright 用例红,而不是让 app 崩。
func installE2EFakes(ctx context.Context) {
	agentruntime.RegisterRuntime(agent_backend_entity.TypeClaudeCode, fakert.New())

	// 幂等:正常每次 e2e run 用全新 AGENTRE_DATA_DIR(临时目录),但 wails dev 热重载
	// 会重启 app 进程,backend 可能已存在 —— 命中则复用,避免重名报错后跳过挂载。
	const backendName = "E2E Local Backend"
	var backendID int64
	if existing, err := agent_backend_repo.AgentBackend().FindByName(ctx, backendName); err != nil {
		logger.Ctx(ctx).Error("main.installE2EFakes: lookup backend failed", zap.Error(err))
		return
	} else if existing != nil {
		backendID = existing.ID
	} else {
		resp, err := agent_backend_svc.AgentBackend().Create(ctx, &agent_backend_svc.CreateBackendRequest{
			Type: string(agent_backend_entity.TypeClaudeCode),
			Name: backendName,
		})
		if err != nil {
			logger.Ctx(ctx).Error("main.installE2EFakes: create backend failed", zap.Error(err))
			return
		}
		backendID = resp.Item.ID
	}

	ceo, err := agent_repo.Agent().FindSystem(ctx)
	if err != nil {
		logger.Ctx(ctx).Error("main.installE2EFakes: find system agent failed", zap.Error(err))
		return
	}
	if ceo == nil {
		logger.Ctx(ctx).Error("main.installE2EFakes: system agent not found (migration gap?)")
		return
	}

	if _, err := agent_svc.Agent().Update(ctx, &agent_svc.UpdateAgentRequest{
		ID:             ceo.ID,
		Name:           ceo.Name,
		AgentBackendID: backendID,
	}); err != nil {
		logger.Ctx(ctx).Error("main.installE2EFakes: attach backend to agent failed", zap.Error(err))
		return
	}

	logger.Ctx(ctx).Info("main.installE2EFakes: e2e fakes installed",
		zap.Int64("backendID", backendID), zap.Int64("agentID", ceo.ID))
}
