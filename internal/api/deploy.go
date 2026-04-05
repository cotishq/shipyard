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

var allowedRepoHosts = map[string]struct{}{
	"github.com": {},
}

var allowedBuildPresets = map[string]string{
	"static-copy": "true",
	"npm":         "npm ci && npm run build",
	"vite":        "npm ci && npm run build",
	"next-export": "npm ci && npm run build && npm run export",
}

type DeployRequest struct {
	ProjectID string `json:"project_id"`
}

func CreateDeployment(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		req := new(DeployRequest)

		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request",
			})
		}
		req.ProjectID = strings.TrimSpace(req.ProjectID)
		if req.ProjectID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "project_id is required",
			})
		}

		var (
			repoURL     string
			buildPreset string
			outputDir   string
		)
		err = db.QueryRow(`
			SELECT repo_url, build_preset, output_dir
			FROM projects
			WHERE id = $1 AND user_id = $2 AND is_active = TRUE
			LIMIT 1
		`, req.ProjectID, userID).Scan(&repoURL, &buildPreset, &outputDir)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "project not found",
				})
			}
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "failed to resolve project configuration",
			})
		}

		config := &ProjectCreateRequest{
			RepoURL:     repoURL,
			BuildPreset: buildPreset,
			OutputDir:   outputDir,
		}
		buildCommand, err := resolveBuildCommand(config)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		id := uuid.New().String()

		_, err = db.Exec(`
		INSERT INTO deployments (id, project_id, repo_url, build_command, output_dir, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		`, id, req.ProjectID, config.RepoURL, buildCommand, config.OutputDir, "QUEUED")

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

type ProjectCreateRequest struct {
	Name          string `json:"name"`
	RepoURL       string `json:"repo_url"`
	BuildPreset   string `json:"build_preset"`
	OutputDir     string `json:"output_dir"`
	DefaultBranch string `json:"default_branch"`
}

func resolveBuildCommand(req *ProjectCreateRequest) (string, error) {
	req.RepoURL = strings.TrimSpace(req.RepoURL)
	req.BuildPreset = strings.TrimSpace(req.BuildPreset)
	req.OutputDir = strings.TrimSpace(req.OutputDir)

	if req.RepoURL == "" {
		return "", errors.New("repo_url is required")
	}
	if strings.ContainsAny(req.RepoURL, " \t\r\n") {
		return "", errors.New("repo_url must not contain whitespace")
	}
	u, err := url.Parse(req.RepoURL)
	if err != nil || u.Host == "" {
		return "", errors.New("repo_url must be a valid URL")
	}
	if u.Scheme != "https" {
		return "", errors.New("repo_url must use https")
	}
	if err := validateRepoHost(u.Hostname()); err != nil {
		return "", err
	}

	if req.BuildPreset == "" {
		return "", errors.New("build_preset is required")
	}

	buildCommand, ok := allowedBuildPresets[req.BuildPreset]
	if !ok {
		return "", errors.New("unsupported build_preset")
	}

	if req.BuildPreset == "static-copy" && req.OutputDir == "" {
		return buildCommand, nil
	}

	if req.OutputDir == "" {
		return "", errors.New("output_dir is required")
	}

	cleaned := path.Clean(req.OutputDir)
	if cleaned == "." {
		req.OutputDir = ""
		return buildCommand, nil
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", errors.New("output_dir must be a relative path")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", errors.New("output_dir must not escape repository root")
	}
	req.OutputDir = cleaned
	return buildCommand, nil
}

func validateProjectCreateRequest(req *ProjectCreateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.DefaultBranch = strings.TrimSpace(req.DefaultBranch)

	if req.Name == "" {
		return errors.New("name is required")
	}
	if len(req.Name) > 120 {
		return errors.New("name is too long")
	}
	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}
	if strings.ContainsAny(req.DefaultBranch, " \t\r\n") {
		return errors.New("default_branch must not contain whitespace")
	}

	_, err := resolveBuildCommand(req)
	if err != nil {
		return err
	}
	return nil
}

func validateDeployRequest(req *ProjectCreateRequest) (string, error) {
	return resolveBuildCommand(req)
}

func validateRepoHost(host string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if _, ok := allowedRepoHosts[host]; !ok {
		return errors.New("repo host is not allowed")
	}
	return nil
}
