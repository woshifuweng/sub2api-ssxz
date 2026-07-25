package response

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

type jsonResponder interface {
	Request() *http.Request
	WriteJSON(status int, payload any)
}

type paginationQueryReader interface {
	QueryValue(name string) string
}

func SuccessContext(c jsonResponder, data any) {
	writeResponse(c, http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func CreatedContext(c jsonResponder, data any) {
	writeResponse(c, http.StatusCreated, Response{Code: 0, Message: "success", Data: data})
}

func AcceptedContext(c jsonResponder, data any) {
	writeResponse(c, http.StatusAccepted, Response{Code: 0, Message: "accepted", Data: data})
}

func ErrorContext(c jsonResponder, statusCode int, message string) {
	writeResponse(c, statusCode, Response{Code: statusCode, Message: message})
}

func ErrorWithDetailsContext(c jsonResponder, statusCode int, message, reason string, metadata map[string]string) {
	writeResponse(c, statusCode, Response{
		Code: statusCode, Message: message, Reason: reason, Metadata: metadata,
	})
}

func ErrorFromContext(c jsonResponder, err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	statusCode, status := infraerrors.ToHTTP(err)
	if statusCode >= 500 {
		if req := c.Request(); req != nil {
			log.Printf("[ERROR] %s %s\n  Error: %s", req.Method, req.URL.Path, logredact.RedactText(err.Error()))
		}
	}
	ErrorWithDetailsContext(c, statusCode, status.Message, status.Reason, status.Metadata)
	return true
}

func PaginatedContext(c jsonResponder, items any, total int64, page, pageSize int) {
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	SuccessContext(c, PaginatedData{Items: items, Total: total, Page: page, PageSize: pageSize, Pages: pages})
}

func PaginatedWithResultContext(c jsonResponder, items any, pagination *PaginationResult) {
	if pagination == nil {
		SuccessContext(c, PaginatedData{Items: items, Page: 1, PageSize: 20, Pages: 1})
		return
	}
	SuccessContext(c, PaginatedData{
		Items: items, Total: pagination.Total, Page: pagination.Page,
		PageSize: pagination.PageSize, Pages: pagination.Pages,
	})
}

func ParsePaginationValues(c paginationQueryReader) (page, pageSize int) {
	page, pageSize = 1, 20
	if p := c.QueryValue("page"); p != "" {
		if value, err := parseInt(p); err == nil && value > 0 {
			page = value
		}
	}
	if value := c.QueryValue("page_size"); value != "" {
		if parsed, err := parseInt(value); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	} else if value := c.QueryValue("limit"); value != "" {
		if parsed, err := parseInt(value); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}
	return page, pageSize
}

func writeResponse(c jsonResponder, status int, payload Response) {
	if c != nil {
		c.WriteJSON(status, payload)
	}
}
