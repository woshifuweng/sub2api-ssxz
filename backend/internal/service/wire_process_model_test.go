package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingletonBackgroundServicesEnabled(t *testing.T) {
	tests := []struct {
		name               string
		role               string
		backgroundServices string
		expected           bool
	}{
		{
			name:               "single-process default enables singleton services",
			role:               "",
			backgroundServices: "",
			expected:           true,
		},
		{
			name:               "coordinator role enables singleton services",
			role:               "coordinator",
			backgroundServices: "",
			expected:           true,
		},
		{
			name:               "coordinator role ignores background env disable",
			role:               " coordinator ",
			backgroundServices: "false",
			expected:           true,
		},
		{
			name:               "worker role disables singleton services by default",
			role:               processRoleWorker,
			backgroundServices: "",
			expected:           false,
		},
		{
			name:               "worker role stays disabled even if background env is true",
			role:               "WORKER",
			backgroundServices: "true",
			expected:           false,
		},
		{
			name:               "master role disables singleton services",
			role:               "master",
			backgroundServices: "true",
			expected:           false,
		},
		{
			name:               "unknown role disables singleton services",
			role:               "sidecar",
			backgroundServices: "on",
			expected:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(processRoleEnvVar, tt.role)
			t.Setenv(backgroundServicesEnvVar, tt.backgroundServices)
			require.Equal(t, tt.expected, singletonBackgroundServicesEnabled())
		})
	}
}

func TestWorkerLocalBackgroundServicesEnabled(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{name: "single process keeps local background services", role: "", expected: true},
		{name: "worker keeps local background services", role: processRoleWorker, expected: true},
		{name: "coordinator does not run worker local services", role: processRoleCoordinator, expected: false},
		{name: "unknown role does not run worker local services", role: "sidecar", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(processRoleEnvVar, tt.role)
			require.Equal(t, tt.expected, workerLocalBackgroundServicesEnabled())
			require.Equal(t, tt.expected, requestPathCacheSyncEnabled())
		})
	}
}

func TestRuntimeAPIRoleDisablesBackgroundServices(t *testing.T) {
	t.Setenv(appRuntimeRoleEnvVar, "api")

	require.True(t, stagingAPIOnlyRuntimeEnabled())
	require.False(t, runtimeBackgroundJobsEnabled())
	require.False(t, runtimeSchedulersEnabled())
	require.False(t, singletonBackgroundServicesEnabled())
	require.False(t, singletonSchedulerServicesEnabled())
	require.False(t, workerLocalBackgroundServicesEnabled())
	require.False(t, requestPathCacheSyncEnabled())
	require.False(t, coordinatorOrSingleProcess())
}

func TestStagingAPIOnlyEnvDisablesBackgroundServices(t *testing.T) {
	t.Setenv(stagingAPIOnlyEnvVar, "true")

	require.True(t, stagingAPIOnlyRuntimeEnabled())
	require.False(t, runtimeBackgroundJobsEnabled())
	require.False(t, runtimeSchedulersEnabled())
	require.False(t, singletonBackgroundServicesEnabled())
	require.False(t, singletonSchedulerServicesEnabled())
	require.False(t, workerLocalBackgroundServicesEnabled())
	require.False(t, requestPathCacheSyncEnabled())
	require.False(t, coordinatorOrSingleProcess())
}

func TestBackgroundJobsDisabledEnvDisablesBackgroundServices(t *testing.T) {
	t.Setenv(backgroundJobsEnvVar, "false")

	require.False(t, stagingAPIOnlyRuntimeEnabled())
	require.False(t, runtimeBackgroundJobsEnabled())
	require.False(t, runtimeSchedulersEnabled())
	require.False(t, singletonBackgroundServicesEnabled())
	require.False(t, singletonSchedulerServicesEnabled())
	require.False(t, workerLocalBackgroundServicesEnabled())
	require.False(t, requestPathCacheSyncEnabled())
	require.False(t, coordinatorOrSingleProcess())
}

func TestSchedulersDisabledEnvOnlyDisablesSchedulerServices(t *testing.T) {
	t.Setenv(schedulersEnvVar, "false")

	require.True(t, runtimeBackgroundJobsEnabled())
	require.False(t, runtimeSchedulersEnabled())
	require.True(t, singletonBackgroundServicesEnabled())
	require.False(t, singletonSchedulerServicesEnabled())
	require.True(t, workerLocalBackgroundServicesEnabled())
	require.True(t, requestPathCacheSyncEnabled())
	require.True(t, coordinatorOrSingleProcess())
}

func TestCoordinatorOrSingleProcess(t *testing.T) {
	t.Setenv(processRoleEnvVar, "")
	require.True(t, coordinatorOrSingleProcess())

	t.Setenv(processRoleEnvVar, processRoleCoordinator)
	require.True(t, coordinatorOrSingleProcess())

	t.Setenv(processRoleEnvVar, processRoleWorker)
	require.False(t, coordinatorOrSingleProcess())
}
