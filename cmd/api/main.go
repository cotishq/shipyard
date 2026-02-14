package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
)


func main() {
	e := echo.New()
	
	e.GET("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "shipyard running")
	})

	log.Println("server running on :8080")
	e.Start(":8080")
}