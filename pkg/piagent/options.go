package piagent

import "time"

type PermissionMode string

const (
	PermissionDefault PermissionMode = "default"
	PermissionPlan    PermissionMode = "plan"
)

const defaultModelID = "gpt-5.5"

type Option func(*Client)

func WithBinary(path string) Option { return func(c *Client) { c.binary = path } }

func WithCwd(path string) Option { return func(c *Client) { c.cwd = path } }

func WithEnv(env map[string]string) Option {
	return func(c *Client) { c.env = cloneMap(env) }
}

func WithModel(model string) Option { return func(c *Client) { c.model = model } }

func WithSystemPrompt(prompt string) Option {
	return func(c *Client) { c.systemPrompt = prompt }
}

func WithThinking(level string) Option {
	return func(c *Client) { c.thinking = level }
}

func WithKillGrace(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.killGrace = d
		}
	}
}

func WithRPCProcessRunnerForTesting(r processRunner) Option {
	return func(c *Client) {
		if r != nil {
			c.runner = r
		}
	}
}
