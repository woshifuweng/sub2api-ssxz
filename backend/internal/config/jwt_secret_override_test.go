//go:build unit

package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyJWTSecretOverride(t *testing.T) {
	productionSecret := strings.Repeat("p", 32)
	overrideSecret := strings.Repeat("s", 32)
	cfg := &Config{JWT: JWTConfig{Secret: productionSecret}}

	t.Setenv("JWT_SECRET_OVERRIDE", overrideSecret)
	t.Setenv("STAGING_MODE", "1")

	require.NoError(t, applyJWTSecretOverride(cfg))
	require.Equal(t, overrideSecret, cfg.JWT.Secret)
}

func TestApplyJWTSecretOverrideRejectsSameSecretInStaging(t *testing.T) {
	productionSecret := strings.Repeat("p", 32)
	cfg := &Config{JWT: JWTConfig{Secret: productionSecret}}

	t.Setenv("JWT_SECRET_OVERRIDE", productionSecret)
	t.Setenv("STAGING_MODE", "1")

	err := applyJWTSecretOverride(cfg)
	require.EqualError(t, err, "staging JWT secret must differ from production")
	require.Equal(t, productionSecret, cfg.JWT.Secret)
}
