package agenttool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	defs := Registry()
	require.Len(t, defs, 1)
	require.Equal(t, "org", defs[0].Key)
	require.Equal(t, "/mcp/org/", defs[0].MCPPath)
	require.Contains(t, defs[0].ToolNames, "org_get")
	require.Len(t, defs[0].ToolNames, 7)

	d, ok := Lookup("org")
	require.True(t, ok)
	require.Equal(t, KeyOrg, d.Key)
	_, ok = Lookup("nope")
	require.False(t, ok)

	require.Equal(t, []string{"org"}, Keys())
}
