package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_Health(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Health(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "linktor", body["service"])
	assert.NotEmpty(t, body["timestamp"])
}

func TestHealthHandler_Ready_NATSDisconnected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// No DB/Redis configured (skipped); NATS reports disconnected -> not ready.
	handler := NewHealthHandler(nil, nil)
	handler.SetNATSChecker(func() bool { return false })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	handler.Ready(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not ready", body["status"])
	checks := body["checks"].(map[string]interface{})
	nats := checks["nats"].(map[string]interface{})
	assert.Equal(t, "unhealthy", nats["status"])
}

func TestHealthHandler_Ready_RedisError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, func(context.Context) error { return errRedisDown })
	handler.SetNATSChecker(func() bool { return true })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	handler.Ready(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHealthHandler_Ready_AllHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Redis ping ok, NATS connected, no DB configured.
	handler := NewHealthHandler(nil, func(context.Context) error { return nil })
	handler.SetNATSChecker(func() bool { return true })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	handler.Ready(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ready", body["status"])
}

var errRedisDown = errRedis("redis down")

type errRedis string

func (e errRedis) Error() string { return string(e) }
