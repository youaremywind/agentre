//go:build e2e

// Package fakes 提供 e2e 构建(`-tags e2e`)专用的确定性 fake 装配。
// 它和 e2e/ 下的 Playwright 工程同处一个目录树,但单独成包,避免 Go 源码与
// TS/Playwright 工具链在同一目录里混在一起。
package fakes

import (
	"context"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"github.com/agentre-ai/agentre/internal/model/entity/agent_backend_entity"
	"github.com/agentre-ai/agentre/internal/pkg/agentruntime"
	fakert "github.com/agentre-ai/agentre/internal/pkg/agentruntime/runtimes/fake"
	"github.com/agentre-ai/agentre/internal/repository/agent_backend_repo"
	"github.com/agentre-ai/agentre/internal/repository/agent_repo"
	"github.com/agentre-ai/agentre/internal/service/agent_backend_svc"
	"github.com/agentre-ai/agentre/internal/service/agent_svc"
)

// Install 仅在 `-tags e2e` 构建中编译:
//  1. 用确定性 fake 覆盖 claudecode runtime(无子进程/无登录);
//  2. seed 一个本地 claudecode backend 并挂到默认 CEO agent,
//     让前端"建会话→发消息→看回复"无需真实 CLI 即可跑通。
//
// 失败只记日志不 panic:e2e 环境异常应让 Playwright 用例红,而不是让 app 崩。
func Install(ctx context.Context) {
	agentruntime.RegisterRuntime(agent_backend_entity.TypeClaudeCode, fakert.New())

	// 幂等:正常每次 e2e run 用全新 AGENTRE_DATA_DIR(临时目录),但 wails dev 热重载
	// 会重启 app 进程,backend 可能已存在 —— 命中则复用,避免重名报错后跳过挂载。
	const backendName = "E2E Local Backend"
	var backendID int64
	if existing, err := agent_backend_repo.AgentBackend().FindByName(ctx, backendName); err != nil {
		logger.Ctx(ctx).Error("e2efakes.Install: lookup backend failed", zap.Error(err))
		return
	} else if existing != nil {
		backendID = existing.ID
	} else {
		resp, err := agent_backend_svc.AgentBackend().Create(ctx, &agent_backend_svc.CreateBackendRequest{
			Type: string(agent_backend_entity.TypeClaudeCode),
			Name: backendName,
		})
		if err != nil {
			logger.Ctx(ctx).Error("e2efakes.Install: create backend failed", zap.Error(err))
			return
		}
		backendID = resp.Item.ID
	}

	ceo, err := agent_repo.Agent().FindSystem(ctx)
	if err != nil {
		logger.Ctx(ctx).Error("e2efakes.Install: find system agent failed", zap.Error(err))
		return
	}
	if ceo == nil {
		logger.Ctx(ctx).Error("e2efakes.Install: system agent not found (migration gap?)")
		return
	}

	if _, err := agent_svc.Agent().Update(ctx, &agent_svc.UpdateAgentRequest{
		ID:             ceo.ID,
		Name:           ceo.Name,
		AgentBackendID: backendID,
	}); err != nil {
		logger.Ctx(ctx).Error("e2efakes.Install: attach backend to agent failed", zap.Error(err))
		return
	}

	// seed 第二个 agent 当群成员(挂 CEO 汇报线;子 agent 与建群弹窗 eligible 池同口径)
	// —— 任务卡 e2e(group-task.spec)需要群里有人可派活。幂等同 backend:命中名字即复用。
	const memberName = "E2E Member"
	if existing, err := agent_repo.Agent().FindByName(ctx, memberName); err != nil {
		logger.Ctx(ctx).Error("e2efakes.Install: lookup member agent failed", zap.Error(err))
		return
	} else if existing == nil {
		if _, err := agent_svc.Agent().Create(ctx, &agent_svc.CreateAgentRequest{
			Name:           memberName,
			ParentAgentID:  ceo.ID,
			AgentBackendID: backendID,
		}); err != nil {
			logger.Ctx(ctx).Error("e2efakes.Install: create member agent failed", zap.Error(err))
			return
		}
	}

	logger.Ctx(ctx).Info("e2efakes.Install: e2e fakes installed",
		zap.Int64("backendID", backendID), zap.Int64("agentID", ceo.ID))
}
