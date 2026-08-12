package service

import (
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

func setOpsUpstreamRequestBodyAny(ctx any, body []byte) {
	switch value := ctx.(type) {
	case *gin.Context:
		setOpsUpstreamRequestBody(value, body)
	case gatewayctx.GatewayContext:
		setOpsUpstreamRequestBodyContext(value, body)
	}
}

func appendOpsUpstreamErrorAny(ctx any, event OpsUpstreamErrorEvent) {
	switch value := ctx.(type) {
	case *gin.Context:
		appendOpsUpstreamError(value, event)
	case gatewayctx.GatewayContext:
		appendOpsUpstreamErrorContext(value, event)
	}
}

func setOpsUpstreamErrorAny(ctx any, statusCode int, message, detail string) {
	switch value := ctx.(type) {
	case *gin.Context:
		setOpsUpstreamError(value, statusCode, message, detail)
	case gatewayctx.GatewayContext:
		setOpsUpstreamErrorContext(value, statusCode, message, detail)
	}
}
