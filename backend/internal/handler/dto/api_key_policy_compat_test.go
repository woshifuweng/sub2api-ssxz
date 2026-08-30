package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyFromServicePreservesPolicyCompatibilityFields(t *testing.T) {
	primary := int64(10)
	key := &service.APIKey{
		GroupID:       &primary,
		GroupIDs:      []int64{10, 20},
		AllowedModels: []string{"gpt-5.4"},
		Group:         &service.Group{ID: 10, Name: "primary"},
		Groups:        []*service.Group{{ID: 10, Name: "primary"}, {ID: 20, Name: "fallback"}},
	}

	got := APIKeyFromService(key)
	require.Equal(t, &primary, got.GroupID)
	require.Equal(t, []int64{10, 20}, got.GroupIDs)
	require.Equal(t, []string{"gpt-5.4"}, got.AllowedModels)
	require.Len(t, got.Groups, 2)
}
