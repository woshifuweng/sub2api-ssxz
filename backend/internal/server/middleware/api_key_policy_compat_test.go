package middleware

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyMultiGroupAuthorizationValidatesEveryBinding(t *testing.T) {
	groupA := &service.Group{ID: 10, Status: service.StatusActive, IsExclusive: true}
	groupB := &service.Group{ID: 20, Status: service.StatusActive, IsExclusive: true}
	key := &service.APIKey{
		GroupIDs: []int64{10, 20},
		Groups:   []*service.Group{groupA, groupB},
		User:     &service.User{AllowedGroups: []int64{10}},
	}

	require.False(t, validateAPIKeyGroupAllowed(key), "one unauthorized binding must reject the whole key")
	key.User.AllowedGroups = []int64{10, 20}
	require.True(t, validateAPIKeyGroupAllowed(key))
}

func TestAPIKeyMultiGroupAuthorizationFailsClosedWhenHydrationIsIncomplete(t *testing.T) {
	key := &service.APIKey{
		GroupIDs: []int64{10, 20},
		Groups:   []*service.Group{{ID: 10, Status: service.StatusActive}},
		User:     &service.User{},
	}

	_, _, available := validateAPIKeyGroupAvailable(key)
	require.False(t, available)
}

func TestAPIKeyMultiGroupAvailabilityRejectsAnyUnavailableBinding(t *testing.T) {
	tests := []struct {
		name     string
		groups   []*service.Group
		wantCode string
	}{
		{
			name: "disabled secondary group",
			groups: []*service.Group{
				{ID: 10, Status: service.StatusActive},
				{ID: 20, Status: service.StatusDisabled},
			},
			wantCode: "GROUP_DISABLED",
		},
		{
			name: "deleted secondary group",
			groups: []*service.Group{
				{ID: 10, Status: service.StatusActive},
				{ID: 20, Status: "deleted"},
			},
			wantCode: "GROUP_DELETED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &service.APIKey{
				GroupIDs: []int64{10, 20},
				Groups:   tt.groups,
			}

			code, _, available := validateAPIKeyGroupAvailable(key)
			require.False(t, available)
			require.Equal(t, tt.wantCode, code)
		})
	}
}
