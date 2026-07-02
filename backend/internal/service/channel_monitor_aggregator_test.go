package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type channelMonitorUserViewRepo struct {
	ChannelMonitorRepository

	enabledMonitors []*ChannelMonitor
	monitorsByID    map[int64]*ChannelMonitor
	latestByID      map[int64][]*ChannelMonitorLatest

	listLatestIDs []int64
}

func (r *channelMonitorUserViewRepo) ListEnabled(context.Context) ([]*ChannelMonitor, error) {
	return r.enabledMonitors, nil
}

func (r *channelMonitorUserViewRepo) GetByID(_ context.Context, id int64) (*ChannelMonitor, error) {
	if monitor := r.monitorsByID[id]; monitor != nil {
		return monitor, nil
	}
	return nil, ErrChannelMonitorNotFound
}

func (r *channelMonitorUserViewRepo) ListLatestForMonitorIDs(_ context.Context, ids []int64) (map[int64][]*ChannelMonitorLatest, error) {
	r.listLatestIDs = append([]int64{}, ids...)
	return r.latestByID, nil
}

func (r *channelMonitorUserViewRepo) ComputeAvailabilityForMonitors(_ context.Context, ids []int64, _ int) (map[int64][]*ChannelMonitorAvailability, error) {
	out := make(map[int64][]*ChannelMonitorAvailability, len(ids))
	for _, id := range ids {
		out[id] = []*ChannelMonitorAvailability{{
			Model:             "gpt-4o-mini",
			AvailabilityPct:   99.5,
			OperationalChecks: 199,
			TotalChecks:       200,
		}}
	}
	return out, nil
}

func (r *channelMonitorUserViewRepo) ListRecentHistoryForMonitors(
	_ context.Context,
	ids []int64,
	_ map[int64]string,
	_ int,
) (map[int64][]*ChannelMonitorHistoryEntry, error) {
	out := make(map[int64][]*ChannelMonitorHistoryEntry, len(ids))
	now := time.Now().UTC()
	for _, id := range ids {
		out[id] = []*ChannelMonitorHistoryEntry{{
			Status:    MonitorStatusOperational,
			CheckedAt: now,
		}}
	}
	return out, nil
}

func (r *channelMonitorUserViewRepo) ListLatestPerModel(_ context.Context, monitorID int64) ([]*ChannelMonitorLatest, error) {
	return r.latestByID[monitorID], nil
}

func (r *channelMonitorUserViewRepo) ComputeAvailability(_ context.Context, monitorID int64, windowDays int) ([]*ChannelMonitorAvailability, error) {
	return []*ChannelMonitorAvailability{{
		Model:             "gpt-4o-mini",
		WindowDays:        windowDays,
		AvailabilityPct:   99.5,
		OperationalChecks: 199,
		TotalChecks:       200,
	}}, nil
}

func TestChannelMonitorListUserViewFiltersToVisibleGroups(t *testing.T) {
	repo := &channelMonitorUserViewRepo{
		enabledMonitors: []*ChannelMonitor{
			{ID: 1, Name: "Visible monitor", Provider: "openai", GroupName: "Pro", PrimaryModel: "gpt-4o-mini", Enabled: true},
			{ID: 2, Name: "Hidden monitor", Provider: "openai", GroupName: "Internal", PrimaryModel: "gpt-4o-mini", Enabled: true},
			{ID: 3, Name: "Ungrouped monitor", Provider: "openai", GroupName: "", PrimaryModel: "gpt-4o-mini", Enabled: true},
		},
		latestByID: map[int64][]*ChannelMonitorLatest{
			1: {{Model: "gpt-4o-mini", Status: MonitorStatusOperational}},
		},
	}
	svc := NewChannelMonitorService(repo, nil)

	views, err := svc.ListUserView(context.Background(), map[string]struct{}{"Pro": {}})

	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Equal(t, "Visible monitor", views[0].Name)
	require.Equal(t, "Pro", views[0].GroupName)
	require.Equal(t, []int64{1}, repo.listLatestIDs)
}

func TestChannelMonitorGetUserDetailRejectsInvisibleMonitor(t *testing.T) {
	repo := &channelMonitorUserViewRepo{
		monitorsByID: map[int64]*ChannelMonitor{
			7: {ID: 7, Name: "Internal monitor", Provider: "openai", GroupName: "Internal", PrimaryModel: "gpt-4o-mini", Enabled: true},
		},
	}
	svc := NewChannelMonitorService(repo, nil)

	_, err := svc.GetUserDetail(context.Background(), 7, map[string]struct{}{"Pro": {}})

	require.Error(t, err)
	require.True(t, infraerrors.IsNotFound(err))
}

func TestChannelMonitorGetUserDetailAllowsVisibleMonitor(t *testing.T) {
	repo := &channelMonitorUserViewRepo{
		monitorsByID: map[int64]*ChannelMonitor{
			9: {ID: 9, Name: "Pro monitor", Provider: "openai", GroupName: "Pro", PrimaryModel: "gpt-4o-mini", Enabled: true},
		},
		latestByID: map[int64][]*ChannelMonitorLatest{
			9: {{Model: "gpt-4o-mini", Status: MonitorStatusOperational}},
		},
	}
	svc := NewChannelMonitorService(repo, nil)

	detail, err := svc.GetUserDetail(context.Background(), 9, map[string]struct{}{"Pro": {}})

	require.NoError(t, err)
	require.Equal(t, "Pro monitor", detail.Name)
	require.Equal(t, "Pro", detail.GroupName)
	require.Len(t, detail.Models, 1)
	require.Equal(t, "gpt-4o-mini", detail.Models[0].Model)
}
