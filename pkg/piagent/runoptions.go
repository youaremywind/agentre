package piagent

type RunOption func(*runSpec)

type runSpec struct {
	resumeID       string
	permissionMode PermissionMode
}

func Resume(sessionID string) RunOption {
	return func(s *runSpec) { s.resumeID = sessionID }
}

func RunPermissionMode(mode PermissionMode) RunOption {
	return func(s *runSpec) { s.permissionMode = mode }
}
