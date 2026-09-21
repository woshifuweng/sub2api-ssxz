package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestClampProfitControlThreshold 钉死阈值清洗口径：非有限或为负一律按 0
// （D > 0 时才可放行免费上游），合法值原样保留。
func TestClampProfitControlThreshold(t *testing.T) {
	require.Equal(t, 0.0, clampProfitControlThreshold(math.NaN()))
	require.Equal(t, 0.0, clampProfitControlThreshold(math.Inf(1)))
	require.Equal(t, 0.0, clampProfitControlThreshold(math.Inf(-1)))
	require.Equal(t, 0.0, clampProfitControlThreshold(-0.5))
	require.Equal(t, 0.0, clampProfitControlThreshold(0))
	require.Equal(t, 0.7, clampProfitControlThreshold(0.7))
}

// TestProfitControlOverThreshold 钉死折扣阈值比较：相对 epsilon 吸收
// decimal(10,4) 落库与浮点乘法的边界误差，U == 阈值不越线。
//
// 线上与预演通过 profitControlRejectsRate 共用本函数，并另外要求 U < D。
func TestProfitControlOverThreshold(t *testing.T) {
	require.False(t, profitControlOverThreshold(0.7, 0.7), "U == 折扣阈值不越线；准入还需 U < D")
	require.False(t, profitControlOverThreshold(0.6999, 0.7))
	require.True(t, profitControlOverThreshold(0.7001, 0.7))

	// 0.3*3 的浮点尾数不得被判成越线。
	require.False(t, profitControlOverThreshold(0.1*3, 0.3))

	// 阈值为 0 时只放行 0 成本上游。
	require.False(t, profitControlOverThreshold(0, 0))
	require.True(t, profitControlOverThreshold(0.0001, 0))
}
