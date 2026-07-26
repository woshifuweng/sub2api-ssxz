//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAPIKeyAuthSnapshotPreservesExclusiveGroupAdmission 锁定独占分组准入字段
// （Group.IsExclusive / User.AllowedGroups）必须在认证缓存快照中往返存活。
//
// 背景：这两个字段由 1a86c6ce1 引入，在 e5c51dce9（v0.1.163 上游整合）中被
// 上游基座覆盖丢失，导致缓存命中路径上 CanBindGroup(group.ID, group.IsExclusive)
// 第二参恒为 false——独占分组校验在网关热路径上静默失效。
//
// 测试走完整链路：snapshotFromAPIKey → JSON 序列化往返（模拟 L2 Redis 存取）
// → applyAuthCacheEntry 重建，最后按中间件 validateAPIKeyGroupAllowed 的
// 真实调用形态断言 CanBindGroup 的结果。
func TestAPIKeyAuthSnapshotPreservesExclusiveGroupAdmission(t *testing.T) {
	const groupID = int64(9)

	cases := []struct {
		name          string
		isExclusive   bool
		allowedGroups []int64
		wantAllowed   bool
	}{
		{
			name:          "非独占分组：任何用户放行",
			isExclusive:   false,
			allowedGroups: nil,
			wantAllowed:   true,
		},
		{
			name:          "独占分组且用户已授权：放行",
			isExclusive:   true,
			allowedGroups: []int64{7, groupID},
			wantAllowed:   true,
		},
		{
			name:          "独占分组且用户未授权：拒绝",
			isExclusive:   true,
			allowedGroups: []int64{7, 8},
			wantAllowed:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &APIKeyService{}
			gid := groupID

			source := &APIKey{
				ID:      1,
				UserID:  2,
				GroupID: &gid,
				Status:  StatusActive,
				User: &User{
					ID:            2,
					Status:        StatusActive,
					Role:          RoleUser,
					Balance:       10,
					Concurrency:   3,
					AllowedGroups: tc.allowedGroups,
				},
				Group: &Group{
					ID:               groupID,
					Name:             "exclusive-pool",
					Platform:         PlatformOpenAI,
					IsExclusive:      tc.isExclusive,
					Status:           StatusActive,
					SubscriptionType: SubscriptionTypeStandard,
					RateMultiplier:   1,
				},
				Groups: []*Group{{
					ID:          groupID,
					Name:        "exclusive-pool",
					Platform:    PlatformOpenAI,
					IsExclusive: tc.isExclusive,
					Status:      StatusActive,
				}},
			}

			snapshot := svc.snapshotFromAPIKey(context.Background(), source)
			require.NotNil(t, snapshot)
			require.Equal(t, tc.isExclusive, snapshot.Group.IsExclusive,
				"快照构造必须携带 Group.IsExclusive")
			require.Equal(t, tc.allowedGroups, snapshot.User.AllowedGroups,
				"快照构造必须携带 User.AllowedGroups")

			// 模拟 L2 Redis 存取：JSON 序列化往返，锁住字段的 json tag
			raw, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: snapshot})
			require.NoError(t, err)
			var entry APIKeyAuthCacheEntry
			require.NoError(t, json.Unmarshal(raw, &entry))

			rebuilt, ok, err := svc.applyAuthCacheEntry("k-exclusive", &entry)
			require.NoError(t, err)
			require.True(t, ok)
			require.NotNil(t, rebuilt.User)
			require.NotNil(t, rebuilt.Group)

			require.Equal(t, tc.isExclusive, rebuilt.Group.IsExclusive,
				"缓存重建后的 Group.IsExclusive 必须保真")
			require.Equal(t, tc.allowedGroups, rebuilt.User.AllowedGroups,
				"缓存重建后的 User.AllowedGroups 必须保真")
			require.Len(t, rebuilt.Groups, 1)
			require.Equal(t, tc.isExclusive, rebuilt.Groups[0].IsExclusive,
				"多分组快照中的 IsExclusive 必须保真")

			// 与 middleware.validateAPIKeyGroupAllowed 相同的调用形态
			got := rebuilt.User.CanBindGroup(rebuilt.Group.ID, rebuilt.Group.IsExclusive)
			require.Equal(t, tc.wantAllowed, got)
		})
	}
}

// TestAPIKeyAuthSnapshotStaleVersionTreatedAsMiss 锁定版本闸门：旧 schema 缓存条目
// （版本不符，含无 version 字段反序列化为 0 的历史条目）必须按 miss 回源，
// 不得以零值字段继续通过认证判断。
func TestAPIKeyAuthSnapshotStaleVersionTreatedAsMiss(t *testing.T) {
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-stale-version", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  apiKeyAuthSnapshotVersion - 1,
			APIKeyID: 1,
			UserID:   2,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:     2,
				Status: StatusActive,
			},
		},
	})

	require.NoError(t, err)
	require.False(t, ok, "旧版本快照必须按缓存 miss 处理")
	require.Nil(t, apiKey)
}
