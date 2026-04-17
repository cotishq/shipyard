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
	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/metrics"
	"github.com/cotishq/shipyard/internal/observability"
	"github.com/cotishq/shipyard/internal/storage"
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	db.Init()

	storage.Init()
	metrics.Init()

	e := echo.New()

	e.GET("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "shipyard running")
	})

	e.GET("/metrics", func(c *echo.Context) error {
		promhttp.Handler().ServeHTTP(c.Response(), c.Request())
		return nil
	})

	e.GET("/healthz", api.GetHealth)

	limitPerMinute := 60
	if raw := strings.TrimSpace(os.Getenv("SHIPYARD_RATE_LIMIT_PER_MINUTE")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limitPerMinute = parsed
		}
	}

	secured := e.Group("")
	secured.Use(api.APIKeyAuthMiddleware(db.DB))
	secured.Use(api.RateLimitMiddleware(api.NewInMemoryRateLimiter(limitPerMinute, time.Minute)))
	secured.GET("/logs/:id", api.GetLogs)
	secured.POST("/deploy", api.CreateDeployment(db.DB))
	secured.GET("/deployments", api.GetDeployments)
	secured.GET("/deployments/:id", api.GetDeployment)
	secured.POST("/projects", api.CreateProject(db.DB))
	secured.GET("/projects", api.GetProjects(db.DB))
	secured.GET("/projects/:id", api.GetProject(db.DB))
	secured.POST("/projects/:id/deployments", api.TriggerProjectDeployment(db.DB))
	secured.POST("/tokens", api.CreateToken(db.DB))
	secured.GET("/tokens", api.ListTokens(db.DB))
	secured.DELETE("/tokens/:id", api.RevokeToken(db.DB))
	secured.POST("/projects/:id/webhook", api.CreateProjectWebhook(db.DB))

	e.GET("/:id", api.ServeDeployment)
	e.GET("/:id/*", api.ServeDeployment)

	e.Static("/deployments", "/tmp")
	e.POST("/webhooks/github", api.HandleGitHubWebhook(db.DB))

	observability.Info("api server starting", map[string]any{
		"address": ":8082",
	})
	if err := e.Start(":8082"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("failed to start api server:", err)
	}
}
