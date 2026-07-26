//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// balanceHistoryAdminStub 内嵌 AdminService 接口占位，只覆写 GetUserBalanceHistory。
type balanceHistoryAdminStub struct {
	service.AdminService

	gotUserID   int64
	gotPage     int
	gotPageSize int
	gotType     string
}

func (s *balanceHistoryAdminStub) GetUserBalanceHistory(ctx context.Context, userID int64, page, pageSize int, codeType string) ([]service.RedeemCode, int64, float64, error) {
	s.gotUserID = userID
	s.gotPage = page
	s.gotPageSize = pageSize
	s.gotType = codeType
	usedBy := userID
	return []service.RedeemCode{
		{ID: 1, Code: "CODE-1", Type: "balance", Value: 25, Status: service.StatusUsed, UsedBy: &usedBy},
		{ID: -3, Code: "AFF-3", Type: service.RedeemTypeAffiliateBalance, Value: 1.5, Status: service.StatusUsed, UsedBy: &usedBy},
	}, 27, 125.5, nil
}

func TestUserHandlerBalanceHistoryUnauthenticated401(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &UserHandler{adminService: &balanceHistoryAdminStub{}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/balance-history", nil)

	handler.GetBalanceHistory(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestUserHandlerBalanceHistoryReturnsMergedLedger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &balanceHistoryAdminStub{}
	handler := &UserHandler{adminService: stub}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/balance-history?page=2&page_size=10", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.GetBalanceHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	// userID 必须来自 JWT，而不是任何请求参数。
	require.EqualValues(t, 42, stub.gotUserID)
	require.Equal(t, 2, stub.gotPage)
	require.Equal(t, 10, stub.gotPageSize)
	require.Equal(t, "", stub.gotType)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID   int64  `json:"id"`
				Code string `json:"code"`
				Type string `json:"type"`
			} `json:"items"`
			Total          int64   `json:"total"`
			Page           int     `json:"page"`
			PageSize       int     `json:"page_size"`
			Pages          int     `json:"pages"`
			TotalRecharged float64 `json:"total_recharged"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 2)
	require.Equal(t, "CODE-1", resp.Data.Items[0].Code)
	require.Equal(t, "AFF-3", resp.Data.Items[1].Code)
	require.Equal(t, service.RedeemTypeAffiliateBalance, resp.Data.Items[1].Type)
	require.EqualValues(t, 27, resp.Data.Total)
	require.Equal(t, 2, resp.Data.Page)
	require.Equal(t, 10, resp.Data.PageSize)
	require.Equal(t, 3, resp.Data.Pages)
	require.InDelta(t, 125.5, resp.Data.TotalRecharged, 1e-9)
}

func TestUserHandlerBalanceHistoryServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &UserHandler{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/balance-history", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.GetBalanceHistory(c)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
