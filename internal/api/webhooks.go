package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type webhookCreateResponse struct {
	WebhookID string `json:"webhook_id"`
	Secret    string `json:"secret"`
	Endpoint  string `json:"endpoint"`
}

func CreateProjectWebhook(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		projectID := strings.TrimSpace(c.Param("id"))
		if projectID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "project id is required"})
		}

		// verify project ownership
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM projects WHERE id = $1 AND user_id = $2
			)
		`, projectID, userID).Scan(&exists)
		if err != nil || !exists {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "project not found"})
		}

		secret, err := generateRawToken()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate secret"})
		}

		webhookID := uuid.NewString()
		_, err = db.Exec(`
			INSERT INTO project_webhooks (id, project_id, provider, secret, is_active)
			VALUES ($1, $2, 'github', $3, TRUE)
			ON CONFLICT (project_id, provider) DO UPDATE
			SET secret = EXCLUDED.secret,
			    is_active = TRUE
		`, webhookID, projectID, secret)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create webhook"})
		}

		return c.JSON(http.StatusOK, webhookCreateResponse{
			WebhookID: webhookID,
			Secret:    secret,
			Endpoint:  "/webhooks/github",
		})
	}
}

func HandleGitHubWebhook(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		event := c.Request().Header.Get("X-GitHub-Event")
		deliveryID := c.Request().Header.Get("X-GitHub-Delivery")

		if event == "" || deliveryID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing github headers"})
		}

		// raw body for signature verification
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		// parse minimal payload
		var payload struct {
			Repository struct {
				CloneURL string `json:"clone_url"`
			} `json:"repository"`
			Ref string `json:"ref"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		}

		repoURL := strings.TrimSpace(payload.Repository.CloneURL)
		branch := strings.TrimPrefix(payload.Ref, "refs/heads/")

		// lookup project by repo_url + branch
		var projectID, defaultBranch string
		err = db.QueryRow(`
			SELECT id, default_branch
			FROM projects
			WHERE repo_url = $1 AND is_active = TRUE
			LIMIT 1
		`, repoURL).Scan(&projectID, &defaultBranch)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "no matching project"})
		}

		// check branch filter
		if branch != defaultBranch {
			return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
		}

		// load secret
		var secret string
		err = db.QueryRow(`
			SELECT secret FROM project_webhooks
			WHERE project_id = $1 AND provider = 'github' AND is_active = TRUE
		`, projectID).Scan(&secret)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "webhook not configured"})
		}

		if !verifyGitHubSignature(secret, body, c.Request().Header.Get("X-Hub-Signature-256")) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
		}

		// dedupe
		_, err = db.Exec(`
			INSERT INTO webhook_events (id, provider, delivery_id, project_id, event_type)
			VALUES ($1, 'github', $2, $3, $4)
			ON CONFLICT (provider, delivery_id) DO NOTHING
		`, uuid.NewString(), deliveryID, projectID, event)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to record event"})
		}

		// trigger deployment
		deploymentID, err := triggerDeploymentForProject(db, projectID, "")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create deployment"})
		}
		return c.JSON(http.StatusOK, map[string]string{"deployment_id": deploymentID})
	}
}
func verifyGitHubSignature(secret string, body []byte, sigHeader string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	sig := strings.TrimPrefix(sigHeader, prefix)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) == 1
}
func triggerDeploymentForProject(db *sql.DB, projectID, userID string) (string, error) {
	var (
		repoURL       string
		buildPreset   string
		outputDir     string
		defaultBranch string
	)
	var err error
	if strings.TrimSpace(userID) == "" {
		err = db.QueryRow(`
			SELECT repo_url, build_preset, output_dir, default_branch
			FROM projects
			WHERE id = $1 AND is_active = TRUE
			LIMIT 1
		`, projectID).Scan(&repoURL, &buildPreset, &outputDir, &defaultBranch)
	} else {
		err = db.QueryRow(`
			SELECT repo_url, build_preset, output_dir, default_branch
			FROM projects
			WHERE id = $1 AND user_id = $2 AND is_active = TRUE
			LIMIT 1
		`, projectID, userID).Scan(&repoURL, &buildPreset, &outputDir, &defaultBranch)
	}
	if err != nil {
		return "", err
	}

	cfg := &ProjectCreateRequest{
		RepoURL:     repoURL,
		BuildPreset: buildPreset,
		OutputDir:   outputDir,
	}

	buildCommand, err := resolveBuildCommand(cfg)
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(defaultBranch)
	if branch == "" {
		branch = "main"
	}

	deploymentID := uuid.NewString()
	_, err = db.Exec(`
		INSERT INTO deployments (id, project_id, repo_url, build_command, output_dir, branch, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, deploymentID, projectID, cfg.RepoURL, buildCommand, cfg.OutputDir, branch, "QUEUED")
	if err != nil {
		return "", err
	}

	return deploymentID, nil
}
