package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type publicAffiliateSettingRepoStub struct {
	values map[string]string
}

func (s *publicAffiliateSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *publicAffiliateSettingRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *publicAffiliateSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *publicAffiliateSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *publicAffiliateSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *publicAffiliateSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *publicAffiliateSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestGetPublicSettingsExposesAffiliateEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settingService := service.NewSettingService(&publicAffiliateSettingRepoStub{values: map[string]string{
		service.SettingKeyAffiliateEnabled: "true",
	}}, &config.Config{})
	handler := NewSettingHandler(settingService, "test")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil)

	handler.GetPublicSettings(c)

	require.Equal(t, http.StatusOK, w.Code)
	var responseBody struct {
		Data dto.PublicSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &responseBody))
	require.True(t, responseBody.Data.AffiliateEnabled)
}
