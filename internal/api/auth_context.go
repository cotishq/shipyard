package api

import (
	"errors"

	"github.com/labstack/echo/v5"
)

func authenticatedUserID(c *echo.Context) (string, error) {
	v := c.Get("user_id")
	id, ok := v.(string)
	if !ok || id == "" {
		return "", errors.New("missing authenticated user")
	}
	return id, nil
}
