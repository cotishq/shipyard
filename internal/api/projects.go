package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func CreateProject(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		req := new(ProjectCreateRequest)
		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request",
			})
		}

		if err := validateProjectCreateRequest(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		projectID := uuid.NewString()
		_, err = db.Exec(`
			INSERT INTO projects (id, user_id, name, repo_url, build_preset, output_dir, default_branch, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		`, projectID, userID, req.Name, req.RepoURL, req.BuildPreset, req.OutputDir, req.DefaultBranch)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create project",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"project_id": projectID,
		})
	}
}

func GetProjects(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		rows, err := db.Query(`
			SELECT id, user_id, name, repo_url, build_preset, output_dir, default_branch, is_active, created_at
			FROM projects
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to fetch projects",
			})
		}
		defer rows.Close()

		projects := make([]models.Project, 0)
		for rows.Next() {
			var p models.Project
			if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.BuildPreset, &p.OutputDir, &p.DefaultBranch, &p.IsActive, &p.CreatedAt); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "failed to scan project row",
				})
			}
			projects = append(projects, p)
		}

		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to iterate projects",
			})
		}

		return c.JSON(http.StatusOK, projects)
	}
}

func GetProject(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		projectID := strings.TrimSpace(c.Param("id"))
		if projectID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "project id is required",
			})
		}

		var (
			p         models.Project
			createdAt time.Time
		)
		err = db.QueryRow(`
			SELECT id, user_id, name, repo_url, build_preset, output_dir, default_branch, is_active, created_at
			FROM projects
			WHERE id = $1 AND user_id = $2
			LIMIT 1
		`, projectID, userID).Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.BuildPreset, &p.OutputDir, &p.DefaultBranch, &p.IsActive, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "project not found",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to fetch project",
			})
		}

		p.CreatedAt = createdAt
		return c.JSON(http.StatusOK, p)
	}
}

func TriggerProjectDeployment(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		projectID := strings.TrimSpace(c.Param("id"))
		if projectID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "project id is required",
			})
		}

		var (
			repoURL       string
			buildPreset   string
			outputDir     string
			defaultBranch string
		)

		err = db.QueryRow(`
			SELECT repo_url, build_preset, output_dir
			FROM projects
			WHERE id = $1 AND user_id = $2 AND is_active = TRUE
			LIMIT 1
		`, projectID, userID).Scan(&repoURL, &buildPreset, &outputDir, &defaultBranch)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "project not found",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to resolve project configuration",
			})
		}

		cfg := &ProjectCreateRequest{
			RepoURL:     repoURL,
			BuildPreset: buildPreset,
			OutputDir:   outputDir,
		}

		buildCommand, err := resolveBuildCommand(cfg)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		deploymentID := uuid.NewString()

		branch := strings.TrimSpace(defaultBranch)
		if branch == "" {
			branch = "main"
		}

		_, err = db.Exec(`
			INSERT INTO deployments (id, project_id, repo_url, build_command, output_dir, status)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, deploymentID, projectID, cfg.RepoURL, buildCommand, cfg.OutputDir, branch, "QUEUED")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create deployment",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"deployment_id": deploymentID,
		})
	}
}
