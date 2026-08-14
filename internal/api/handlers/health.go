package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db *pgxpool.Pool
	// redisPing, when set, checks Redis connectivity. A function (rather than a
	// concrete client) keeps this handler independent of which go-redis major
	// version the app wires in.
	redisPing func(context.Context) error
	// natsConnected, when set, reports whether the JetStream connection is live.
	// NATS is a hard dependency of the messaging pipeline, so a disconnected
	// broker must fail readiness (otherwise the orchestrator keeps routing to an
	// instance that silently cannot process inbound/outbound traffic).
	natsConnected func() bool
}

// NewHealthHandler creates a new health handler. redisPing may be nil when Redis
// is not configured, in which case readiness skips the Redis check.
func NewHealthHandler(db *pgxpool.Pool, redisPing func(context.Context) error) *HealthHandler {
	return &HealthHandler{
		db:        db,
		redisPing: redisPing,
	}
}

// SetNATSChecker registers a liveness probe for the NATS/JetStream connection.
// Optional: when unset, readiness does not include a NATS check.
func (h *HealthHandler) SetNATSChecker(connected func() bool) {
	h.natsConnected = connected
}

// version reports which build is answering, para que o CD consiga distinguir
// "a API respondeu" de "a versão nova subiu".
//
// Sem isto, o smoke test do pipeline consultava /health logo após despachar o
// deploy e recebia 200 do container ANTIGO, passando em 0,4s: ele nunca
// verificou o rollout, e um deploy que não aconteceu passava como sucesso.
//
// Vazio quando não configurado — é informação de operação, não contrato: quem
// consome trata a ausência como "desconhecido", nunca como falha.
func version() string {
	return os.Getenv("LINKTOR_VERSION")
}

// Health godoc
// @Summary      Health check
// @Description  Returns basic health status of the service
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200 {object} object{status=string,service=string,version=string,timestamp=string}
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "linktor",
		"version":   version(),
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
	if h.redisPing != nil {
		if err := h.redisPing(ctx); err != nil {
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

	// Check NATS/JetStream — a disconnected broker means the messaging pipeline
	// is down, so readiness must fail even though the HTTP layer is up.
	if h.natsConnected != nil {
		if h.natsConnected() {
			checks["nats"] = map[string]interface{}{"status": "healthy"}
		} else {
			checks["nats"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  "not connected",
			}
			allHealthy = false
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
