package service

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

// buildChatStreamErrorSSE keeps streaming clients on the OpenAI error envelope.
func buildChatStreamErrorSSE(code, message string) string {
	payload, err := json.Marshal(gin.H{
		"error": gin.H{
			"type":    "invalid_request_error",
			"code":    code,
			"message": message,
		},
	})
	if err != nil {
		return "data: {\"error\":{\"type\":\"invalid_request_error\",\"code\":\"" + code + "\",\"message\":\"upstream error\"}}\n\n"
	}
	return "data: " + string(payload) + "\n\n"
}
