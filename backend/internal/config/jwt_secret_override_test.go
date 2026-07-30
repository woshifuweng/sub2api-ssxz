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
	cfg := &Config{JWT: JWTConfig{Secret: productionSecret, Env: "staging"}}

	t.Setenv("JWT_SECRET_OVERRIDE", overrideSecret)

	require.NoError(t, applyJWTSecretOverride(cfg))
	require.Equal(t, overrideSecret, cfg.JWT.Secret)
}

func TestApplyJWTSecretOverrideRejectsSameSecretInNonProduction(t *testing.T) {
	productionSecret := strings.Repeat("p", 32)
	for _, source := range []string{"JWT_SECRET_OVERRIDE", "JWT_SECRET", "SSXZ_JWT_SECRET"} {
		t.Run(source, func(t *testing.T) {
			cfg := &Config{JWT: JWTConfig{Secret: productionSecret, Env: "staging"}}
			t.Setenv("JWT_SECRET_OVERRIDE", "")
			t.Setenv("JWT_SECRET", "")
			t.Setenv("SSXZ_JWT_SECRET", "")
			t.Setenv(source, productionSecret)

			err := applyJWTSecretOverride(cfg)
			require.EqualError(
				t,
				err,
				`JWT secret for env="staging" must differ from production secret (source=`+source+`)`,
			)
			require.Equal(t, productionSecret, cfg.JWT.Secret)
		})
	}
}
