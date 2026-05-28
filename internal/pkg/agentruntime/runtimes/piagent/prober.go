package piagent

import (
	"agentre/internal/model/entity/agent_backend_entity"
	"agentre/internal/pkg/agentruntime"
)

// DefaultBinary returns the executable name used when cli_path is empty.
func DefaultBinary() string { return "pi" }

const fallbackModelID = "gpt-5.5"

func defaultModelForBackend(b *agent_backend_entity.AgentBackend) string {
	if b != nil && b.ReasoningEffort != "" {
		return fallbackModelID + ":" + b.ReasoningEffort
	}
	return fallbackModelID
}

func BuildPiAgentEnv(b *agent_backend_entity.AgentBackend) (map[string]string, error) {
	return agentruntime.BuildPiAgentEnv(b)
}
