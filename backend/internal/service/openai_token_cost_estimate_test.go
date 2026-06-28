package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayServiceEstimateOpenAITokenRequestCost(t *testing.T) {
	svc := newOpenAIRecordUsageServiceForTest(nil, nil, nil, nil)
	t.Cleanup(svc.CloseOpenAIWSPool)

	chatCost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.1",
		[]byte(`{"model":"gpt-5.1","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":200000}`),
		&APIKey{},
		&User{ID: 1},
	)
	require.NoError(t, err)
	require.NotNil(t, chatCost)
	require.Greater(t, chatCost.ActualCost, 2.0)

	responsesCost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.1",
		[]byte(`{"model":"gpt-5.1","input":"hello","max_output_tokens":1000}`),
		&APIKey{},
		&User{ID: 1},
	)
	require.NoError(t, err)
	require.NotNil(t, responsesCost)
	require.Greater(t, responsesCost.ActualCost, 0.01)

	noLimitCost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.1",
		[]byte(`{"model":"gpt-5.1","input":"hello"}`),
		&APIKey{},
		&User{ID: 1},
	)
	require.NoError(t, err)
	require.Nil(t, noLimitCost)
}
