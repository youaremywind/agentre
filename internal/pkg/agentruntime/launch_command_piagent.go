package agentruntime

import (
	"fmt"
	"strings"

	"agentre/internal/model/entity/agent_backend_entity"
)

func buildPiAgentShellCommand(spec LaunchCommandSpec, cwd string) (string, error) {
	env, err := BuildPiAgentEnv(spec.Backend)
	if err != nil {
		return "", err
	}
	binary := strings.TrimSpace(spec.Backend.CLIPath)
	if binary == "" {
		binary = "pi"
	}
	argv := []string{binary, "--mode", "rpc", "--no-context-files"}
	if model := piAgentModel(spec.Backend, spec.ProviderSessionID); model != "" {
		argv = append(argv, "--model", model)
	}
	if eff := strings.TrimSpace(spec.Backend.ReasoningEffort); eff != "" {
		argv = append(argv, "--thinking", eff)
	}
	return assembleShellLine(cwd, env, argv), nil
}

func piAgentModel(b *agent_backend_entity.AgentBackend, _ string) string {
	if b == nil {
		return ""
	}
	if eff := strings.TrimSpace(b.ReasoningEffort); eff != "" {
		return fmt.Sprintf("gpt-5.5:%s", eff)
	}
	return "gpt-5.5"
}
