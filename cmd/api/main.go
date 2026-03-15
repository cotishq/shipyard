package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/api"
	"github.com/cotishq/shipyard/internal/config"
	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/storage"
	"github.com/labstack/echo/v5"
)

func main() {
	db.Init()

	storage.Init()

	e := echo.New()

	e.GET("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "shipyard running")
	})

	e.GET("/healthz", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	apiKey := strings.TrimSpace(os.Getenv("SHIPYARD_API_KEY"))
	if err := config.ValidateAPIKey(apiKey); err != nil && !config.AllowInsecureDefaults() {
		log.Fatal(err)
	}

	limitPerMinute := 60
	if raw := strings.TrimSpace(os.Getenv("SHIPYARD_RATE_LIMIT_PER_MINUTE")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limitPerMinute = parsed
		}
	}

	secured := e.Group("")
	secured.Use(api.APIKeyAuthMiddleware(apiKey))
	secured.Use(api.RateLimitMiddleware(api.NewInMemoryRateLimiter(limitPerMinute, time.Minute)))
	secured.GET("/logs/:id", api.GetLogs)
	secured.POST("/deploy", api.CreateDeployment(db.DB))
	secured.GET("/deployments", api.GetDeployments)
	secured.GET("/deployments/:id", api.GetDeployment)

	e.GET("/:id", api.ServeDeployment)
	e.GET("/:id/*", api.ServeDeployment)

	e.Static("/deployments", "/tmp")

	log.Println("server running successfully on :8082")
	if err := e.Start(":8082"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("failed to start api server:", err)
	}
}
