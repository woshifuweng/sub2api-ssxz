package service

import (
	"os"
	"testing"
)

func TestBackgroundJobsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		set     bool
		value   string
		enabled bool
	}{
		{name: "unset preserves production default", enabled: true},
		{name: "explicit true", set: true, value: "true", enabled: true},
		{name: "explicit false", set: true, value: "false", enabled: false},
		{name: "whitespace is accepted", set: true, value: " 0 ", enabled: false},
		{name: "invalid value fails closed", set: true, value: "disabled-ish", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv(backgroundJobsEnabledEnv, tt.value)
			} else {
				oldValue, wasSet := os.LookupEnv(backgroundJobsEnabledEnv)
				if err := os.Unsetenv(backgroundJobsEnabledEnv); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if wasSet {
						_ = os.Setenv(backgroundJobsEnabledEnv, oldValue)
					} else {
						_ = os.Unsetenv(backgroundJobsEnabledEnv)
					}
				})
			}
			if got := backgroundJobsEnabled(); got != tt.enabled {
				t.Fatalf("backgroundJobsEnabled() = %v, want %v", got, tt.enabled)
			}
		})
	}
}

func TestStartBackgroundJobRespectsSwitch(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		t.Setenv(backgroundJobsEnabledEnv, "false")
		started := 0
		startBackgroundJob(func() { started++ })
		if started != 0 {
			t.Fatalf("disabled background job started %d times", started)
		}
	})

	t.Run("enabled", func(t *testing.T) {
		t.Setenv(backgroundJobsEnabledEnv, "true")
		started := 0
		startBackgroundJob(func() { started++ })
		if started != 1 {
			t.Fatalf("enabled background job started %d times", started)
		}
	})
}
