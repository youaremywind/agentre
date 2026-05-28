package piagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRPCArgsIncludesSystemPromptAndThinking(t *testing.T) {
	args := buildRPCArgs(&Client{
		cwd:          "/tmp/session",
		model:        "gpt-5.5",
		thinking:     "high",
		systemPrompt: "be concise",
	})

	assert.Equal(t, []string{
		"--mode", "rpc",
		"--session-dir", "/tmp/session",
		"--append-system-prompt", "be concise",
		"--model", "gpt-5.5",
		"--thinking", "high",
		"--no-context-files",
	}, args)
}

func TestBuildRPCArgsClampsUnsupportedMaxThinking(t *testing.T) {
	args := buildRPCArgs(&Client{thinking: "max"})
	assert.Equal(t, []string{"--mode", "rpc", "--thinking", "xhigh", "--no-context-files"}, args)
}

func TestBuildRPCArgsOmitsInvalidThinking(t *testing.T) {
	args := buildRPCArgs(&Client{thinking: "ultra"})
	assert.Equal(t, []string{"--mode", "rpc", "--no-context-files"}, args)
}
