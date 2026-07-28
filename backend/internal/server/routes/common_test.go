package routes

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterCommonRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterCommonRoutes(router)

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
	})

	t.Run("event logging batch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/event_logging/batch", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("setup status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"code":0,"data":{"needs_setup":false,"step":"completed"}}`, rec.Body.String())
	})

	t.Run("readyz is loopback-only and returns schema state", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		readyzRouter := gin.New()
		RegisterCommonRoutes(readyzRouter, db)

		req := httptest.NewRequest(http.MethodGet, "/internal/readyz", nil)
		req.RemoteAddr = "10.0.0.5:12345"
		forbidden := httptest.NewRecorder()
		readyzRouter.ServeHTTP(forbidden, req)
		require.Equal(t, http.StatusForbidden, forbidden.Code)

		mock.ExpectQuery(regexp.QuoteMeta(readyzSchemaQuery)).
			WillReturnRows(sqlmock.NewRows([]string{"count", "latest"}).AddRow(int64(3), "190_latest.sql"))
		req = httptest.NewRequest(http.MethodGet, "/internal/readyz", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		ok := httptest.NewRecorder()
		readyzRouter.ServeHTTP(ok, req)
		require.Equal(t, http.StatusOK, ok.Code)
		require.JSONEq(t, `{"schema_migrations_count":3,"schema_migrations_latest_filename":"190_latest.sql","status":"ok"}`, ok.Body.String())
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
