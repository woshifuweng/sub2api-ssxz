package admin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountImportUploadSessionAcceptsTotalBytesAtLimit(t *testing.T) {
	manager := &accountImportUploadSessionManager{sessions: make(map[string]*accountImportUploadSession)}
	state, filePath, err := manager.createSession(createAccountImportUploadSessionRequest{
		Filename:   "accounts.json",
		TotalBytes: accountImportUploadMaxTotalBytes,
	})
	require.NoError(t, err)
	require.NotNil(t, state)
	t.Cleanup(func() { manager.deleteSession(state.SessionID) })

	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, accountImportUploadMaxTotalBytes, info.Size())
	require.Equal(t, accountImportUploadMaxTotalBytes, state.TotalBytes)
}

func TestAccountImportUploadSessionRejectsTotalBytesAboveLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &accountImportUploadSessionManager{sessions: make(map[string]*accountImportUploadSession)}
	handler := &AccountHandler{uploadSessionManager: manager}
	router := gin.New()
	router.POST("/upload", handler.CreateImportUploadSession)
	request := httptest.NewRequest(
		http.MethodPost,
		"/upload",
		strings.NewReader(fmt.Sprintf(`{"filename":"accounts.json","total_bytes":%d}`, accountImportUploadMaxTotalBytes+1)),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "total_bytes exceeds maximum")
	manager.mu.RLock()
	sessionCount := len(manager.sessions)
	manager.mu.RUnlock()
	require.Zero(t, sessionCount)
}

func TestAccountImportUploadChunksNeverExceedDeclaredTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &accountImportUploadSessionManager{sessions: make(map[string]*accountImportUploadSession)}
	state, filePath, err := manager.createSession(createAccountImportUploadSessionRequest{
		Filename:   "accounts.json",
		TotalBytes: 5,
	})
	require.NoError(t, err)
	t.Cleanup(func() { manager.deleteSession(state.SessionID) })

	handler := &AccountHandler{uploadSessionManager: manager}
	router := gin.New()
	router.PUT("/upload/:session_id", handler.UploadImportChunk)
	upload := func(offset int64, body string, contentLength int64) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/upload/%s?offset=%d", state.SessionID, offset),
			strings.NewReader(body),
		)
		request.ContentLength = contentLength
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	require.Equal(t, http.StatusOK, upload(0, "abc", 3).Code)
	session, ok := manager.getSession(state.SessionID)
	require.True(t, ok)
	require.Equal(t, int64(3), session.state.ReceivedBytes)
	require.Equal(t, accountImportUploadStatusUploading, session.state.Status)

	require.Equal(t, http.StatusBadRequest, upload(3, "def", -1).Code)
	session, ok = manager.getSession(state.SessionID)
	require.True(t, ok)
	require.Equal(t, int64(3), session.state.ReceivedBytes)
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, int64(5), info.Size())

	require.Equal(t, http.StatusOK, upload(3, "de", 2).Code)
	session, ok = manager.getSession(state.SessionID)
	require.True(t, ok)
	require.Equal(t, int64(5), session.state.ReceivedBytes)
	require.Equal(t, accountImportUploadStatusUploaded, session.state.Status)
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, []byte("abcde"), data)
}
