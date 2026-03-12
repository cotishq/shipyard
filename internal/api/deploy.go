package api

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DeployRequest struct {
	RepoURL      string `json:"repo_url"`
	BuildCommand string `json:"build_command"`
	OutputDir    string `json:"output_dir"`
}

func CreateDeployment(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		req := new(DeployRequest)

		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request",
			})
		}
		if err := validateDeployRequest(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
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

func validateDeployRequest(req *DeployRequest) error {
	req.RepoURL = strings.TrimSpace(req.RepoURL)
	req.BuildCommand = strings.TrimSpace(req.BuildCommand)
	req.OutputDir = strings.TrimSpace(req.OutputDir)

	if req.RepoURL == "" {
		return errors.New("repo_url is required")
	}
	if strings.ContainsAny(req.RepoURL, " \t\r\n") {
		return errors.New("repo_url must not contain whitespace")
	}
	u, err := url.Parse(req.RepoURL)
	if err != nil || u.Host == "" {
		return errors.New("repo_url must be a valid URL")
	}
	if u.Scheme != "https" {
		return errors.New("repo_url must use https")
	}

	if req.BuildCommand == "" {
		return errors.New("build_command is required")
	}
	if len(req.BuildCommand) > 500 {
		return errors.New("build_command is too long")
	}

	if req.OutputDir == "" {
		return nil
	}

	cleaned := path.Clean(req.OutputDir)
	if cleaned == "." {
		req.OutputDir = ""
		return nil
	}
	if strings.HasPrefix(cleaned, "/") {
		return errors.New("output_dir must be a relative path")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return errors.New("output_dir must not escape repository root")
	}
	req.OutputDir = cleaned
	return nil
}
