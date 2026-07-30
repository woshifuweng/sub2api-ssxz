package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWT_StagingTokenRejectedByProduction(t *testing.T) {
	staging := newJWTEnvironmentTestService("staging")
	production := newJWTEnvironmentTestService("production")

	token, err := staging.GenerateToken(context.Background(), jwtEnvironmentTestUser())
	require.NoError(t, err)

	claims, err := production.ValidateToken(token)
	require.Error(t, err)
	require.Nil(t, claims)
}

func TestJWT_ProductionTokenRejectedByStaging(t *testing.T) {
	production := newJWTEnvironmentTestService("production")
	staging := newJWTEnvironmentTestService("staging")

	token, err := production.GenerateToken(context.Background(), jwtEnvironmentTestUser())
	require.NoError(t, err)

	claims, err := staging.ValidateToken(token)
	require.Error(t, err)
	require.Nil(t, claims)
}

func TestJWT_LegacyTokenWithoutIssuerAccepted(t *testing.T) {
	production := newJWTEnvironmentTestService("production")
	now := time.Now()
	legacyClaims := &JWTClaims{
		UserID:       1,
		Email:        "legacy@test.local",
		Role:         RoleUser,
		TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyClaims).
		SignedString([]byte(production.cfg.JWT.Secret))
	require.NoError(t, err)

	claims, err := production.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.Empty(t, claims.Issuer)
}

func TestJWT_NewTokenHasIssuerClaim(t *testing.T) {
	production := newJWTEnvironmentTestService("production")

	token, err := production.GenerateToken(context.Background(), jwtEnvironmentTestUser())
	require.NoError(t, err)

	claims, err := production.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, "production", claims.Issuer)
}

func newJWTEnvironmentTestService(env string) *AuthService {
	return &AuthService{
		cfg: &config.Config{
			JWT: config.JWTConfig{
				Secret:     "jwt-environment-test-secret-32-bytes",
				Env:        env,
				ExpireHour: 1,
			},
		},
	}
}

func jwtEnvironmentTestUser() *User {
	return &User{
		ID:           1,
		Email:        "jwt-env@test.local",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}
}
