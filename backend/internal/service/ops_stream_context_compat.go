package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

const (
	OpsStreamErrorKey    = "ops_stream_error"
	ResponseCommittedKey = "response_committed"

	OpsClientBusinessLimitedKey                          = "ops_client_business_limited"
	OpsClientBusinessLimitedReasonKey                    = "ops_client_business_limited_reason"
	OpsClientBusinessLimitedReasonIPRestriction          = "api_key_ip_restriction"
	OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable = "api_key_group_unavailable"
	OpsClientBusinessLimitedReasonAPIKeyGroupUnassigned  = "api_key_group_unassigned"
	OpsClientBusinessLimitedReasonLocalFeatureGate       = "local_feature_gate"
	OpsClientBusinessLimitedReasonLocalPolicyDenied      = "local_policy_denied"
)

func MarkResponseCommitted(c *gin.Context) { c.Set(ResponseCommittedKey, true) }

func IsResponseCommitted(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(ResponseCommittedKey)
	if !ok {
		return false
	}
	marked, _ := v.(bool)
	return marked
}

func MarkOpsClientBusinessLimited(c *gin.Context, reason string) {
	if c == nil {
		return
	}
	c.Set(OpsClientBusinessLimitedKey, true)
	if reason = strings.TrimSpace(reason); reason != "" {
		c.Set(OpsClientBusinessLimitedReasonKey, reason)
	}
}

func MarkOpsClientBusinessLimitedAny(c any, reason string) {
	switch value := c.(type) {
	case *gin.Context:
		MarkOpsClientBusinessLimited(value, reason)
	case gatewayctx.GatewayContext:
		if value == nil {
			return
		}
		value.SetValue(OpsClientBusinessLimitedKey, true)
		if reason = strings.TrimSpace(reason); reason != "" {
			value.SetValue(OpsClientBusinessLimitedReasonKey, reason)
		}
	}
}

func HasOpsClientBusinessLimited(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(OpsClientBusinessLimitedKey)
	if !ok {
		return false
	}
	marked, _ := v.(bool)
	return marked
}

type OpsStreamError struct {
	ErrType         string
	Code            string
	Message         string
	IntendedStatus  int
	CountTowardsSLA bool
}

func MarkOpsStreamError(c *gin.Context, errType, message string, intendedStatus int) {
	markOpsStreamError(c, OpsStreamError{ErrType: errType, Message: message, IntendedStatus: intendedStatus})
}

func MarkOpsStreamFailure(c *gin.Context, errType, code, message string, intendedStatus int) {
	markOpsStreamError(c, OpsStreamError{
		ErrType: errType, Code: code, Message: message,
		IntendedStatus: intendedStatus, CountTowardsSLA: true,
	})
}

func markOpsStreamError(c *gin.Context, streamErr OpsStreamError) {
	if c == nil {
		return
	}
	if _, exists := c.Get(OpsStreamErrorKey); exists {
		return
	}
	streamErr.ErrType = strings.TrimSpace(streamErr.ErrType)
	streamErr.Code = strings.TrimSpace(streamErr.Code)
	streamErr.Message = strings.TrimSpace(streamErr.Message)
	c.Set(OpsStreamErrorKey, streamErr)
}

func GetOpsStreamError(c *gin.Context) (OpsStreamError, bool) {
	if c == nil {
		return OpsStreamError{}, false
	}
	v, ok := c.Get(OpsStreamErrorKey)
	if !ok {
		return OpsStreamError{}, false
	}
	streamErr, ok := v.(OpsStreamError)
	return streamErr, ok
}
