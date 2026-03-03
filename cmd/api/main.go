package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/cotishq/shipyard/internal/api"
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

	e.GET("/logs/:id", api.GetLogs)
	e.POST("/deploy", api.CreateDeployment(db.DB))
	e.GET("/deployments/:id", api.GetDeployment)
	e.GET("/:id", api.ServeDeployment)
	e.GET("/:id/*", api.ServeDeployment)

	e.Static("/deployments", "/tmp")

	log.Println("server running successfully on :8082")
	if err := e.Start(":8082"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("failed to start api server:", err)
	}
}
