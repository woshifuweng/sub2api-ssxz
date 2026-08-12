package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	proxyRequestFailureCooldown  = 10 * time.Minute
	proxyRequestQuickFailTimeout = 5 * time.Second
)

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnCloseReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if c.cancel != nil {
		c.cancel()
	}
	return err
}

func newProxyRequestFailoverError(account *Account, proxyURL string, err error) *UpstreamFailoverError {
	failoverErr := &UpstreamFailoverError{
		StatusCode:           http.StatusBadGateway,
		ResponseBody:         []byte(sanitizeUpstreamErrorMessage(err.Error())),
		TempUnscheduleFor:    proxyRequestFailureCooldown,
		TempUnscheduleReason: "upstream request failed via proxy/network (auto temp-unschedule 10m)",
		FailedProxyURL:       strings.TrimSpace(proxyURL),
	}
	if account != nil && account.ProxyID != nil && *account.ProxyID > 0 {
		failoverErr.FailedProxyID = *account.ProxyID
	}
	return failoverErr
}

func withProxyQuickFailRequest(req *http.Request, proxyURL string) (*http.Request, context.CancelFunc) {
	if req == nil || strings.TrimSpace(proxyURL) == "" {
		return req, nil
	}
	if deadline, ok := req.Context().Deadline(); ok && time.Until(deadline) <= proxyRequestQuickFailTimeout {
		return req, nil
	}
	ctx, cancel := context.WithTimeout(req.Context(), proxyRequestQuickFailTimeout)
	return req.WithContext(ctx), cancel
}

func attachProxyQuickFailCancel(resp *http.Response, cancel context.CancelFunc) *http.Response {
	if cancel == nil {
		return resp
	}
	if resp == nil || resp.Body == nil {
		cancel()
		return resp
	}
	resp.Body = &cancelOnCloseReadCloser{ReadCloser: resp.Body, cancel: cancel}
	return resp
}
