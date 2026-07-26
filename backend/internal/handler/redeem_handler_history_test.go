//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// redeemHistoryRepoStub 只覆写 ListByUserPaginated；其余接口方法由内嵌接口占位，
// 被意外调用时直接 panic，保证测试只走分页路径。
type redeemHistoryRepoStub struct {
	service.RedeemCodeRepository

	gotUserID int64
	gotParams pagination.PaginationParams
	codes     []service.RedeemCode
	total     int64
}

func (s *redeemHistoryRepoStub) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	s.gotUserID = userID
	s.gotParams = params
	pages := int((s.total + int64(params.Limit()) - 1) / int64(params.Limit()))
	if pages < 1 {
		pages = 1
	}
	return s.codes, &pagination.PaginationResult{Total: s.total, Page: params.Page, PageSize: params.Limit(), Pages: pages}, nil
}

func newRedeemHistoryHandler(repo service.RedeemCodeRepository) *RedeemHandler {
	return NewRedeemHandler(service.NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil), nil)
}

func TestRedeemHandlerGetHistoryUnauthenticated401(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := newRedeemHistoryHandler(&redeemHistoryRepoStub{})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/redeem/history", nil)

	handler.GetHistory(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestRedeemHandlerGetHistoryPaginates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &redeemHistoryRepoStub{
		codes: []service.RedeemCode{
			{ID: 7, Code: "CODE-7", Type: "balance", Value: 25, Status: service.StatusUsed},
		},
		total: 42,
	}
	handler := newRedeemHistoryHandler(repo)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/redeem/history?page=3&page_size=5", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 11})

	handler.GetHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.EqualValues(t, 11, repo.gotUserID)
	require.Equal(t, 3, repo.gotParams.Page)
	require.Equal(t, 5, repo.gotParams.PageSize)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID   int64  `json:"id"`
				Code string `json:"code"`
			} `json:"items"`
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
			Pages    int   `json:"pages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, "CODE-7", resp.Data.Items[0].Code)
	require.EqualValues(t, 42, resp.Data.Total)
	require.Equal(t, 3, resp.Data.Page)
	require.Equal(t, 5, resp.Data.PageSize)
	require.Equal(t, 9, resp.Data.Pages)
}

func TestRedeemHandlerGetHistoryDefaultsPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &redeemHistoryRepoStub{total: 0}
	handler := newRedeemHistoryHandler(repo)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/redeem/history", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 11})

	handler.GetHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, repo.gotParams.Page)
	require.Equal(t, 20, repo.gotParams.PageSize)
}
