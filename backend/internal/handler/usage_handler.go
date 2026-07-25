package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UsageHandler handles usage-related requests
type UsageHandler struct {
	usageService   *service.UsageService
	apiKeyService  *service.APIKeyService
	opsService     *service.OpsService
	settingService *service.SettingService
}

type usageGatewayResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g usageGatewayResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g usageGatewayResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}

// NewUsageHandler creates a new UsageHandler
func NewUsageHandler(
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	opsService *service.OpsService,
	settingService *service.SettingService,
) *UsageHandler {
	return &UsageHandler{
		usageService:   usageService,
		apiKeyService:  apiKeyService,
		opsService:     opsService,
		settingService: settingService,
	}
}

// List handles listing usage records with pagination
// GET /api/v1/usage
func (h *UsageHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	parsed, ok := h.parseUserUsageFiltersGateway(c, false)
	if !ok {
		return
	}
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    defaultQueryValue(c, "sort_by", "created_at"),
		SortOrder: defaultQueryValue(c, "sort_order", "desc"),
	}
	records, result, err := h.usageService.ListWithFilters(c.Request().Context(), params, parsed.Filters)
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	out := make([]dto.UsageLog, 0, len(records))
	for i := range records {
		out = append(out, *dto.UsageLogFromService(&records[i]))
	}
	response.PaginatedContext(usageGatewayResponder{ctx: c}, out, result.Total, page, pageSize)
}

// GetByID handles getting a single usage record
// GET /api/v1/usage/:id
func (h *UsageHandler) GetByID(c *gin.Context) {
	h.GetByIDGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) GetByIDGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	usageID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid usage ID")
		return
	}

	record, err := h.usageService.GetByID(c.Request().Context(), usageID)
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	// 验证所有权
	if record.UserID != subject.UserID {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusForbidden, "Not authorized to access this record")
		return
	}

	response.SuccessContext(usageGatewayResponder{ctx: c}, dto.UsageLogFromService(record))
}

// Stats handles getting usage statistics
// GET /api/v1/usage/stats
func (h *UsageHandler) Stats(c *gin.Context) {
	h.StatsGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) StatsGateway(c gatewayctx.GatewayContext) {
	parsed, ok := h.parseUserUsageFiltersGateway(c, true)
	if !ok {
		return
	}

	stats, err := h.usageService.GetStatsWithFilters(c.Request().Context(), parsed.Filters)
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}
	stats.TotalAccountCost = nil
	stats.UpstreamEndpoints = nil
	stats.EndpointPaths = nil

	response.SuccessContext(usageGatewayResponder{ctx: c}, stats)
}

func (h *UsageHandler) statsGatewayLegacy(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var apiKeyID int64
	if apiKeyIDStr := c.QueryValue("api_key_id"); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid api_key_id")
			return
		}

		// [Security Fix] Verify API Key ownership to prevent horizontal privilege escalation
		apiKey, err := h.apiKeyService.GetByID(c.Request().Context(), id)
		if err != nil {
			response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusNotFound, "API key not found")
			return
		}
		if apiKey.UserID != subject.UserID {
			response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusForbidden, "Not authorized to access this API key's statistics")
			return
		}

		apiKeyID = id
	}

	// 获取时间范围参数
	userTZ := c.QueryValue("timezone")
	now := timezone.NowInUserLocation(userTZ)
	var startTime, endTime time.Time

	// 优先使用 start_date 和 end_date 参数
	startDateStr := c.QueryValue("start_date")
	endDateStr := c.QueryValue("end_date")

	if startDateStr != "" && endDateStr != "" {
		// 使用自定义日期范围
		var err error
		startTime, err = timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		endTime, err = timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		// 与 SQL 条件 created_at < end 对齐，使用次日 00:00 作为上边界（DST-safe）。
		endTime = endTime.AddDate(0, 0, 1)
	} else {
		// 使用 period 参数
		period := defaultQueryValue(c, "period", "today")
		switch period {
		case "today":
			startTime = timezone.StartOfDayInUserLocation(now, userTZ)
		case "week":
			startTime = now.AddDate(0, 0, -7)
		case "month":
			startTime = now.AddDate(0, -1, 0)
		default:
			startTime = timezone.StartOfDayInUserLocation(now, userTZ)
		}
		endTime = now
	}

	var stats *service.UsageStats
	var err error
	if apiKeyID > 0 {
		stats, err = h.usageService.GetStatsByAPIKey(c.Request().Context(), apiKeyID, startTime, endTime)
	} else {
		stats, err = h.usageService.GetStatsByUser(c.Request().Context(), subject.UserID, startTime, endTime)
	}
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(usageGatewayResponder{ctx: c}, stats)
}

// parseUserTimeRange parses start_date, end_date query parameters for user dashboard
// Uses user's timezone if provided, otherwise falls back to server timezone
func parseUserTimeRange(c *gin.Context) (time.Time, time.Time) {
	return parseUserTimeRangeGateway(gatewayctx.FromGin(c))
}

func parseUserTimeRangeGateway(c gatewayctx.GatewayContext) (time.Time, time.Time) {
	userTZ := c.QueryValue("timezone")
	now := timezone.NowInUserLocation(userTZ)
	startDate := c.QueryValue("start_date")
	endDate := c.QueryValue("end_date")

	var startTime, endTime time.Time

	if startDate != "" {
		if t, err := timezone.ParseInUserLocation("2006-01-02", startDate, userTZ); err == nil {
			startTime = t
		} else {
			startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
		}
	} else {
		startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
	}

	if endDate != "" {
		if t, err := timezone.ParseInUserLocation("2006-01-02", endDate, userTZ); err == nil {
			endTime = t.Add(24 * time.Hour) // Include the end date
		} else {
			endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
		}
	} else {
		endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
	}

	return startTime, endTime
}

func defaultQueryValue(c gatewayctx.GatewayContext, key, fallback string) string {
	if c == nil {
		return fallback
	}
	if value := strings.TrimSpace(c.QueryValue(key)); value != "" {
		return value
	}
	return fallback
}

// DashboardStats handles getting user dashboard statistics
// GET /api/v1/usage/dashboard/stats
func (h *UsageHandler) DashboardStats(c *gin.Context) {
	h.DashboardStatsGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) DashboardStatsGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	stats, err := h.usageService.GetUserDashboardStats(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(usageGatewayResponder{ctx: c}, stats)
}

// DashboardTrend handles getting user usage trend data
// GET /api/v1/usage/dashboard/trend
func (h *UsageHandler) DashboardTrend(c *gin.Context) {
	h.DashboardTrendGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) DashboardTrendGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	startTime, endTime := parseUserTimeRangeGateway(c)
	granularity := defaultQueryValue(c, "granularity", "day")

	trend, err := h.usageService.GetUserUsageTrendByUserID(c.Request().Context(), subject.UserID, startTime, endTime, granularity)
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(usageGatewayResponder{ctx: c}, gin.H{
		"trend":       trend,
		"start_date":  startTime.Format("2006-01-02"),
		"end_date":    endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"granularity": granularity,
	})
}

// DashboardModels handles getting user model usage statistics
// GET /api/v1/usage/dashboard/models
func (h *UsageHandler) DashboardModels(c *gin.Context) {
	h.DashboardModelsGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) DashboardModelsGateway(c gatewayctx.GatewayContext) {
	parsed, ok := h.parseUserUsageFiltersGateway(c, true)
	if !ok {
		return
	}

	stats, err := h.usageService.GetModelStatsWithFiltersBySource(c.Request().Context(), parsed.StartTime, parsed.EndTime, parsed.Filters, "requested")
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(usageGatewayResponder{ctx: c}, gin.H{
		"models":     userModelStatsFromUsageStats(stats),
		"start_date": parsed.StartTime.Format("2006-01-02"),
		"end_date":   parsed.EndTime.Add(-24 * time.Hour).Format("2006-01-02"),
	})
}

// BatchAPIKeysUsageRequest represents the request for batch API keys usage
type BatchAPIKeysUsageRequest struct {
	APIKeyIDs []int64 `json:"api_key_ids" binding:"required"`
}

// DashboardAPIKeysUsage handles getting usage stats for user's own API keys
// POST /api/v1/usage/dashboard/api-keys-usage
func (h *UsageHandler) DashboardAPIKeysUsage(c *gin.Context) {
	h.DashboardAPIKeysUsageGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) DashboardAPIKeysUsageGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req BatchAPIKeysUsageRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.APIKeyIDs) == 0 {
		response.SuccessContext(usageGatewayResponder{ctx: c}, gin.H{"stats": map[string]any{}})
		return
	}

	// Limit the number of API key IDs to prevent SQL parameter overflow
	if len(req.APIKeyIDs) > 100 {
		response.ErrorContext(usageGatewayResponder{ctx: c}, http.StatusBadRequest, "Too many API key IDs (maximum 100 allowed)")
		return
	}

	validAPIKeyIDs, err := h.apiKeyService.VerifyOwnership(c.Request().Context(), subject.UserID, req.APIKeyIDs)
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	if len(validAPIKeyIDs) == 0 {
		response.SuccessContext(usageGatewayResponder{ctx: c}, gin.H{"stats": map[string]any{}})
		return
	}

	stats, err := h.usageService.GetBatchAPIKeyUsageStats(c.Request().Context(), validAPIKeyIDs, time.Time{}, time.Time{})
	if err != nil {
		response.ErrorFromContext(usageGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(usageGatewayResponder{ctx: c}, gin.H{"stats": stats})
}
