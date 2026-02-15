package main

import (
	"log"
	"net/http"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/labstack/echo/v5"
)


func main() {
	db.Init()
	e := echo.New()
	
	e.GET("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "shipyard running")
	})

	log.Println("server running on :8080")
	e.Start(":8080")
}