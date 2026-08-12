package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
)

type userUsageFilters struct {
	Filters   usagestats.UsageLogFilters
	StartTime time.Time
	EndTime   time.Time
}

type userModelStat struct {
	Model               string  `json:"model"`
	Requests            int64   `json:"requests"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
	ActualCost          float64 `json:"actual_cost"`
}

type userGroupStat struct {
	GroupID     int64   `json:"group_id"`
	GroupName   string  `json:"group_name"`
	Requests    int64   `json:"requests"`
	TotalTokens int64   `json:"total_tokens"`
	Cost        float64 `json:"cost"`
	ActualCost  float64 `json:"actual_cost"`
}

func (h *UsageHandler) parseUserUsageFilters(c *gin.Context, requireRange bool) (*userUsageFilters, bool) {
	return h.parseUserUsageFiltersGateway(gatewayctx.FromGin(c), requireRange)
}

func (h *UsageHandler) parseUserUsageFiltersGateway(c gatewayctx.GatewayContext, requireRange bool) (*userUsageFilters, bool) {
	responder := usageGatewayResponder{ctx: c}
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(responder, 401, "User not authenticated")
		return nil, false
	}
	if modelSource := strings.TrimSpace(c.QueryValue("model_source")); modelSource != "" && modelSource != usagestats.ModelSourceRequested {
		response.ErrorContext(responder, 400, "model_source is not available for user usage queries")
		return nil, false
	}

	var apiKeyID int64
	if apiKeyIDStr := strings.TrimSpace(c.QueryValue("api_key_id")); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(responder, 400, "Invalid api_key_id")
			return nil, false
		}
		if h.apiKeyService == nil {
			response.ErrorContext(responder, 500, "API key service not available")
			return nil, false
		}
		apiKey, err := h.apiKeyService.GetByID(c.Context(), id)
		if err != nil {
			response.ErrorFromContext(responder, err)
			return nil, false
		}
		if apiKey.UserID != subject.UserID {
			response.ErrorContext(responder, 403, "Not authorized to access this API key's usage records")
			return nil, false
		}
		apiKeyID = id
	}

	var groupID int64
	if groupIDStr := strings.TrimSpace(c.QueryValue("group_id")); groupIDStr != "" {
		id, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(responder, 400, "Invalid group_id")
			return nil, false
		}
		groupID = id
	}

	var requestType *int16
	var stream *bool
	if requestTypeStr := strings.TrimSpace(c.QueryValue("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.ErrorContext(responder, 400, err.Error())
			return nil, false
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := strings.TrimSpace(c.QueryValue("stream")); streamStr != "" {
		val, err := strconv.ParseBool(streamStr)
		if err != nil {
			response.ErrorContext(responder, 400, "Invalid stream value, use true or false")
			return nil, false
		}
		stream = &val
	}

	var billingType *int8
	if billingTypeStr := strings.TrimSpace(c.QueryValue("billing_type")); billingTypeStr != "" {
		val, err := strconv.ParseInt(billingTypeStr, 10, 8)
		if err != nil {
			response.ErrorContext(responder, 400, "Invalid billing_type")
			return nil, false
		}
		bt := int8(val)
		billingType = &bt
	}

	billingMode := strings.TrimSpace(c.QueryValue("billing_mode"))
	if billingMode != "" && !service.BillingMode(billingMode).IsValid() {
		response.ErrorContext(responder, 400, "Invalid billing_mode")
		return nil, false
	}

	userTZ := c.QueryValue("timezone")
	now := timezone.NowInUserLocation(userTZ)
	var startTime, endTime time.Time
	var startPtr, endPtr *time.Time
	startDateStr := strings.TrimSpace(c.QueryValue("start_date"))
	endDateStr := strings.TrimSpace(c.QueryValue("end_date"))

	if startDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.ErrorContext(responder, 400, "Invalid start_date format, use YYYY-MM-DD")
			return nil, false
		}
		startTime = t
		startPtr = &startTime
	}
	if endDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.ErrorContext(responder, 400, "Invalid end_date format, use YYYY-MM-DD")
			return nil, false
		}
		endTime = t.AddDate(0, 0, 1)
		endPtr = &endTime
	}

	if requireRange {
		if startPtr == nil {
			switch defaultQueryValue(c, "period", "") {
			case "today":
				startTime = timezone.StartOfDayInUserLocation(now, userTZ)
			case "week":
				startTime = now.AddDate(0, 0, -7)
			case "month":
				startTime = now.AddDate(0, -1, 0)
			default:
				startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
			}
			startPtr = &startTime
		}
		if endPtr == nil {
			if strings.TrimSpace(c.QueryValue("period")) != "" {
				endTime = now
			} else {
				endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
			}
			endPtr = &endTime
		}
	}

	return &userUsageFilters{
		Filters: usagestats.UsageLogFilters{
			UserID:            subject.UserID,
			APIKeyID:          apiKeyID,
			GroupID:           groupID,
			Model:             strings.TrimSpace(c.QueryValue("model")),
			ModelFilterSource: usagestats.ModelSourceRequested,
			RequestType:       requestType,
			Stream:            stream,
			BillingType:       billingType,
			BillingMode:       billingMode,
			StartTime:         startPtr,
			EndTime:           endPtr,
		},
		StartTime: derefTime(startPtr),
		EndTime:   derefTime(endPtr),
	}, true
}

func userModelStatsFromUsageStats(stats []usagestats.ModelStat) []userModelStat {
	out := make([]userModelStat, 0, len(stats))
	for _, stat := range stats {
		out = append(out, userModelStat{
			Model:               stat.Model,
			Requests:            stat.Requests,
			InputTokens:         stat.InputTokens,
			OutputTokens:        stat.OutputTokens,
			CacheCreationTokens: stat.CacheCreationTokens,
			CacheReadTokens:     stat.CacheReadTokens,
			TotalTokens:         stat.TotalTokens,
			Cost:                stat.Cost,
			ActualCost:          stat.ActualCost,
		})
	}
	return out
}

func userGroupStatsFromUsageStats(stats []usagestats.GroupStat) []userGroupStat {
	out := make([]userGroupStat, 0, len(stats))
	for _, stat := range stats {
		out = append(out, userGroupStat{
			GroupID:     stat.GroupID,
			GroupName:   stat.GroupName,
			Requests:    stat.Requests,
			TotalTokens: stat.TotalTokens,
			Cost:        stat.Cost,
			ActualCost:  stat.ActualCost,
		})
	}
	return out
}

func parseBoolQueryWithDefault(c *gin.Context, key string, fallback bool) (bool, bool) {
	raw := c.Query(key)
	if strings.TrimSpace(raw) == "" {
		return fallback, true
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		response.BadRequest(c, "Invalid "+key+" value, use true or false")
		return false, false
	}
	return parsed, true
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
