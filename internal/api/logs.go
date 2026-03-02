package api

import (
	"net/http"
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/labstack/echo/v5"
)

func GetLogs(c *echo.Context) error {
	id := c.Param("id")

	rows, err := db.DB.Query(`
	SELECT message, created_at
	FROM deployment_logs
	WHERE deployment_id = $1
	ORDER BY created_at
`, id)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch logs",
		})
	}
	defer rows.Close()

	logs := make([]map[string]interface{}, 0)

	for rows.Next() {
		var msg string
		var createdAt time.Time

		if err := rows.Scan(&msg, &createdAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan log row",
			})
		}

		logs = append(logs, map[string]interface{}{
			"message": msg,
			"time":    createdAt.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to iterate logs",
		})
	}

	return c.JSON(http.StatusOK, logs)
}
