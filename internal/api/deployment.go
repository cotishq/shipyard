package api

import (
	"database/sql"
	"net/http"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/labstack/echo/v5"
)


type DeploymentResponse struct {
	ID 			string 			`json:"id"`
	Status 		string 			`json:"status"`
	AttempCount string 			`json:"attemp_count"`
	MaxAttempts string 			`json:"max_attempts"`
	CreatedAt 	string   		`json:"created_at"`
	URL 		string 			`json:"url"`
}

func GetDeployment(c *echo.Context) error {
	id := c.Param("id")

	var resp DeploymentResponse

	err := db.DB.QueryRow(`
	    SELECT id, status, attempt_count, max_attempts, created_at
		FROM deployments
		WHERE id = $1
		`, id).Scan(
			&resp.ID,
			&resp.Status,
			&resp.AttempCount,
			&resp.MaxAttempts,
			&resp.CreatedAt,
		)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "deployment not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch deployment",
		})
	}

	resp.URL = "http://localhost:8001/" + resp.ID

	return c.JSON(http.StatusOK, resp)
}