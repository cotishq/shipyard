package api

import (
	"context"
	"net/http"
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/storage"
	"github.com/labstack/echo/v5"
)

func GetHealth(c *echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	response := map[string]any{
		"status": "ok",
		"services": map[string]string{
			"database": "ok",
			"minio":    "ok",
		},
	}

	statusCode := http.StatusOK

	if err := db.HealthCheck(ctx); err != nil {
		response["status"] = "degraded"
		response["services"].(map[string]string)["database"] = "error"
		response["database_error"] = err.Error()
		statusCode = http.StatusServiceUnavailable
	}

	if err := storage.HealthCheck(ctx); err != nil {
		response["status"] = "degraded"
		response["services"].(map[string]string)["minio"] = "error"
		response["minio_error"] = err.Error()
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, response)
}
