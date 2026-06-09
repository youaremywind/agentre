//go:build e2e

package fake

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"agentre/internal/model/entity/agent_backend_entity"
	"agentre/internal/pkg/agentruntime"
)

func TestRun_EchoesPromptThenDone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r := New()
	events, result, err := r.Run(ctx, agentruntime.RunRequest{
		Backend:   &agent_backend_entity.AgentBackend{ID: 1, Type: string(agent_backend_entity.TypeClaudeCode)},
		SessionID: 42,
		UserText:  "ping",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	var text string
	var sawDone bool
	for ev := range events {
		switch e := ev.(type) {
		case agentruntime.TextDelta:
			text += e.Text
		case agentruntime.Done:
			sawDone = true
		}
	}

	assert.Equal(t, ReplyPrefix+"ping", text)
	assert.True(t, sawDone)
	assert.Equal(t, "e2e-fake-42", result.ProviderSessionID)
	assert.Equal(t, "e2e-fake-model", result.Model)
}

func TestRun_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before draining

	r := New()
	events, _, err := r.Run(ctx, agentruntime.RunRequest{SessionID: 7, UserText: "hello world this is a long enough prompt to span several chunks"})
	require.NoError(t, err)

	// Draining a pre-cancelled run must terminate (channel closes) without hanging.
	for range events { //nolint:revive // draining
	}
}
