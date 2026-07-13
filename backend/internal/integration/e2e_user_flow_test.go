//go:build e2e

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	customerHandoffEnabledEnv  = "CUSTOMER_HANDOFF_E2E"
	customerHandoffEmailEnv    = "CUSTOMER_HANDOFF_EMAIL"
	customerHandoffPasswordEnv = "CUSTOMER_HANDOFF_PASSWORD"
	customerHandoffGroupIDEnv  = "CUSTOMER_HANDOFF_GROUP_ID"
	customerHandoffQuotaEnv    = "CUSTOMER_HANDOFF_KEY_QUOTA"
)

type customerAPIEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type customerLoginData struct {
	AccessToken string `json:"access_token"`
	Requires2FA bool   `json:"requires_2fa"`
}

type customerAPIKey struct {
	ID  int64  `json:"id"`
	Key string `json:"key"`
}

type customerAPIKeyList struct {
	Items []customerAPIKey `json:"items"`
}

type customerModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// TestCustomerHandoffGoldenPath is the strict, no-provider customer handoff gate.
// It creates one temporary low-quota key, verifies local model discovery, then
// deletes the key and proves that the revoked credential immediately returns 401.
func TestCustomerHandoffGoldenPath(t *testing.T) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv(customerHandoffEnabledEnv)), "true") {
		t.Skipf("set %s=true to run the write-enabled customer handoff gate", customerHandoffEnabledEnv)
	}

	email := requireCustomerHandoffEnv(t, customerHandoffEmailEnv)
	password := requireCustomerHandoffEnv(t, customerHandoffPasswordEnv)
	groupID := requirePositiveInt64Env(t, customerHandoffGroupIDEnv)
	quota := customerHandoffQuota(t)
	client := &http.Client{Timeout: 30 * time.Second}

	accessToken := customerLogin(t, client, email, password)

	status, body := customerRequest(t, client, http.MethodGet, "/api/v1/auth/me", nil, accessToken, "")
	requireCustomerStatus(t, "current user", status, http.StatusOK, body, false)

	requestID := fmt.Sprintf("customer-handoff-%d", time.Now().UnixNano())
	createPayload := map[string]any{
		"name":      requestID,
		"group_id":  groupID,
		"group_ids": []int64{groupID},
		"quota":     quota,
	}
	status, body = customerRequest(t, client, http.MethodPost, "/api/v1/keys", createPayload, accessToken, requestID)
	requireCustomerStatus(t, "create API key", status, http.StatusOK, body, true)

	var created customerAPIKey
	decodeCustomerEnvelopeData(t, "create API key", body, &created, true)
	if created.ID <= 0 || strings.TrimSpace(created.Key) == "" || strings.Contains(created.Key, "...") {
		t.Fatal("create API key did not return a one-time plaintext credential")
	}

	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		_, _ = customerRequest(t, client, http.MethodDelete, fmt.Sprintf("/api/v1/keys/%d", created.ID), nil, accessToken, "")
	})

	status, body = customerRequest(t, client, http.MethodGet, "/api/v1/keys?page=1&page_size=100", nil, accessToken, "")
	requireCustomerStatus(t, "list API keys", status, http.StatusOK, body, false)
	var listed customerAPIKeyList
	decodeCustomerEnvelopeData(t, "list API keys", body, &listed, false)
	listedKey, ok := findCustomerAPIKey(listed.Items, created.ID)
	if !ok {
		t.Fatalf("new API key id %d is missing from the list response", created.ID)
	}
	if listedKey.Key == created.Key || !strings.Contains(listedKey.Key, "...") {
		t.Fatal("API key list did not mask the one-time plaintext credential")
	}

	status, body = customerRequest(t, client, http.MethodGet, "/v1/models", nil, created.Key, "")
	requireCustomerStatus(t, "model discovery", status, http.StatusOK, body, false)
	var models customerModelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		t.Fatalf("model discovery returned invalid JSON: %v", err)
	}
	if len(models.Data) == 0 {
		t.Fatal("model discovery returned no models for the selected key group")
	}

	status, body = customerRequest(t, client, http.MethodDelete, fmt.Sprintf("/api/v1/keys/%d", created.ID), nil, accessToken, "")
	requireCustomerStatus(t, "delete API key", status, http.StatusOK, body, false)
	deleted = true

	status, body = customerRequest(t, client, http.MethodGet, "/v1/models", nil, created.Key, "")
	requireCustomerStatus(t, "revoked API key", status, http.StatusUnauthorized, body, false)

	t.Logf("customer handoff gate passed for temporary key id %d with %d discoverable models", created.ID, len(models.Data))
}

func customerLogin(t *testing.T, client *http.Client, email, password string) string {
	t.Helper()
	status, body := customerRequest(t, client, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "", "")
	requireCustomerStatus(t, "login", status, http.StatusOK, body, true)

	var login customerLoginData
	decodeCustomerEnvelopeData(t, "login", body, &login, true)
	if login.Requires2FA {
		t.Fatal("customer handoff test account requires 2FA; use a dedicated low-risk test account")
	}
	if strings.TrimSpace(login.AccessToken) == "" {
		t.Fatal("login response did not contain an access token")
	}
	return login.AccessToken
}

func customerRequest(t *testing.T, client *http.Client, method, path string, payload any, bearerToken, idempotencyKey string) (int, []byte) {
	t.Helper()

	var requestBody io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode %s request: %v", path, err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, requestBody)
	if err != nil {
		t.Fatalf("create %s request: %v", path, err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatalf("read %s response: %v", path, err)
	}
	return response.StatusCode, body
}

func decodeCustomerEnvelopeData(t *testing.T, label string, body []byte, target any, sensitive bool) {
	t.Helper()
	var envelope customerAPIEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		if sensitive {
			t.Fatalf("%s returned an invalid response envelope", label)
		}
		t.Fatalf("%s returned an invalid response envelope: %v", label, err)
	}
	if envelope.Code != 0 {
		t.Fatalf("%s returned API code %d: %s", label, envelope.Code, envelope.Message)
	}
	if len(envelope.Data) == 0 {
		t.Fatalf("%s response did not contain data", label)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		if sensitive {
			t.Fatalf("%s returned invalid data", label)
		}
		t.Fatalf("decode %s data: %v", label, err)
	}
}

func requireCustomerStatus(t *testing.T, label string, actual, expected int, body []byte, sensitive bool) {
	t.Helper()
	if actual == expected {
		return
	}
	if sensitive {
		t.Fatalf("%s returned HTTP %d; expected %d (response redacted)", label, actual, expected)
	}
	message := strings.TrimSpace(string(body))
	if len(message) > 500 {
		message = message[:500] + "..."
	}
	t.Fatalf("%s returned HTTP %d; expected %d: %s", label, actual, expected, message)
}

func requireCustomerHandoffEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s must be set when %s=true", name, customerHandoffEnabledEnv)
	}
	return value
}

func requirePositiveInt64Env(t *testing.T, name string) int64 {
	t.Helper()
	raw := requireCustomerHandoffEnv(t, name)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		t.Fatalf("%s must be a positive integer", name)
	}
	return value
}

func customerHandoffQuota(t *testing.T) float64 {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(customerHandoffQuotaEnv))
	if raw == "" {
		return 1
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 || value > 20 {
		t.Fatalf("%s must be greater than 0 and no more than 20", customerHandoffQuotaEnv)
	}
	return value
}

func findCustomerAPIKey(keys []customerAPIKey, id int64) (customerAPIKey, bool) {
	for _, key := range keys {
		if key.ID == id {
			return key, true
		}
	}
	return customerAPIKey{}, false
}
