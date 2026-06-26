package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateAPIKeyRequestIPRestrictionFieldPresence(t *testing.T) {
	var omitted UpdateAPIKeyRequest
	require.NoError(t, json.Unmarshal([]byte(`{"status":"inactive"}`), &omitted))
	require.Nil(t, omitted.IPWhitelist)
	require.Nil(t, omitted.IPBlacklist)

	var empty UpdateAPIKeyRequest
	require.NoError(t, json.Unmarshal([]byte(`{"ip_whitelist":[],"ip_blacklist":[]}`), &empty))
	require.NotNil(t, empty.IPWhitelist)
	require.NotNil(t, empty.IPBlacklist)
	require.Empty(t, *empty.IPWhitelist)
	require.Empty(t, *empty.IPBlacklist)
}
