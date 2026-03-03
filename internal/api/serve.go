package api

import (
	"net/http"
	"strings"

	"github.com/cotishq/shipyard/internal/storage"
	"github.com/labstack/echo/v5"
)

func ServeDeployment(c *echo.Context) error {
	id := c.Param("id")
	filePath := c.Param("*")

	if filePath == "" {
		filePath = "index.html"
	}

	objectName := id + "/" + filePath

	object, err := storage.Client.GetObject(
		c.Request().Context(),
		"deployments",
		objectName,
		storage.GetOptions(),
	)
	if err != nil {
		return c.String(http.StatusNotFound, "file not found")
	}
	defer object.Close()

	// MinIO GetObject may return a reader before object existence is checked.
	if _, err := object.Stat(); err != nil {
		return c.String(http.StatusNotFound, "file not found")
	}

	contentType := "text/html"
	if strings.HasSuffix(filePath, ".css") {
		contentType = "text/css"
	} else if strings.HasSuffix(filePath, ".js") {
		contentType = "application/javascript"
	} else if strings.HasSuffix(filePath, ".png") {
		contentType = "image/png"
	}

	return c.Stream(http.StatusOK, contentType, object)
}
