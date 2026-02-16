package api

import (
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)


type DeployRequest struct {
	RepoURL 		string `json:"repo_url"`
	BuildCommand 	string `json:"build_command"`
	OutputDir		string `json:"output_dir"`
}

func CreateDeployment(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		req := new(DeployRequest)

		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request",
			})
		}

		id := uuid.New().String()

		_, err := db.Exec(`
		INSERT INTO deployments (id, repo_url, build_command, output_dir, status)
		VALUES ($1, $2, $3, $4, $5)
		`, id, req.RepoURL, req.BuildCommand, req.OutputDir, "QUEUED")

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"deployment_id": id,
		})
	}
}