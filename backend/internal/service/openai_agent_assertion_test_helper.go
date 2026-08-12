package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func decodeAgentAssertionTask(t *testing.T, header string) string {
	t.Helper()
	encoded := strings.TrimPrefix(header, "AgentAssertion ")
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var envelope struct {
		TaskID string `json:"task_id"`
	}
	require.NoError(t, json.Unmarshal(decoded, &envelope))
	return envelope.TaskID
}
