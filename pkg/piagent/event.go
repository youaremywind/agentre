package piagent

import "github.com/cago-frame/agents/provider"

type EventKind string

const (
	EventTextDelta       EventKind = "text_delta"
	EventThinkingDelta   EventKind = "thinking_delta"
	EventPreToolUse      EventKind = "pre_tool_use"
	EventPostToolUse     EventKind = "post_tool_use"
	EventUsage           EventKind = "usage"
	EventCompactBoundary EventKind = "compact_boundary"
	EventRuntimeStatus   EventKind = "runtime_status"
	EventError           EventKind = "error"
	EventDone            EventKind = "done"
)

type Event struct {
	Kind      EventKind
	Text      string
	Tool      ToolEvent
	Usage     provider.Usage
	Model     string
	SessionID string
	Err       error
}

type ToolEvent struct {
	ID      string
	Name    string
	Input   []byte
	Content string
	IsError bool
}
