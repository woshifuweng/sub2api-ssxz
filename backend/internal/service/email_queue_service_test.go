package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEmailQueueServiceWithAutoStartDisabledDoesNotStartWorkers(t *testing.T) {
	svc := NewEmailQueueServiceWithAutoStart(nil, 2, false)
	require.NotNil(t, svc)
	require.Equal(t, 2, svc.workers)

	for i := 0; i < cap(svc.taskChan); i++ {
		svc.taskChan <- EmailTask{Email: "qa@example.test", SiteName: "staging", TaskType: TaskTypeVerifyCode}
	}
	require.Error(t, svc.EnqueueVerifyCode("qa@example.test", "staging"))
}
