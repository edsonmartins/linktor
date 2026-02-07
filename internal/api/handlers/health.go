package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

// Health godoc
// @Summary      Health check
// @Description  Returns basic health status of the service
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200 {object} object{status=string,service=string,timestamp=string}
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "linktor",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready godoc
// @Summary      Readiness check
// @Description  Returns readiness status with dependency checks (PostgreSQL, Redis)
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200 {object} object{status=string,service=string,timestamp=string,checks=object}
// @Failure      503 {object} object{status=string,service=string,timestamp=string,checks=object}
// @Router       /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]interface{})
	allHealthy := true

	// Check PostgreSQL
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["postgres"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			allHealthy = false
		} else {
			checks["postgres"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			allHealthy = false
		} else {
			checks["redis"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	status := http.StatusOK
	statusText := "ready"
	if !allHealthy {
		status = http.StatusServiceUnavailable
		statusText = "not ready"
	}

	c.JSON(status, gin.H{
		"status":    statusText,
		"service":   "linktor",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    checks,
	})
}
