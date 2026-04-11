package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/labstack/echo/v5"
)

type DeploymentResponse struct {
	ID                   string  `json:"id"`
	ProjectID            *string `json:"project_id,omitempty"`
	Branch               string  `json:"branch,omitempty"`
	Status               string  `json:"status"`
	AttempCount          string  `json:"attempt_count"`
	MaxAttempts          string  `json:"max_attempts"`
	CreatedAt            string  `json:"created_at"`
	StartedAt            *string `json:"started_at,omitempty"`
	FinishedAt           *string `json:"finished_at,omitempty"`
	ErrorMessage         *string `json:"error_message,omitempty"`
	BuildDurationSeconds *int    `json:"build_duration_seconds,omitempty"`
	URL                  string  `json:"url"`
}

func GetDeployment(c *echo.Context) error {
	id := c.Param("id")
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	var resp DeploymentResponse
	var (
		projectID            sql.NullString
		startedAt            sql.NullTime
		finishedAt           sql.NullTime
		errorMessage         sql.NullString
		buildDurationSeconds sql.NullInt64
	)

	err = db.DB.QueryRow(`
		SELECT d.id, d.project_id, d.branch, d.status, d.attempt_count, d.max_attempts, d.created_at, d.started_at, d.finished_at, d.error_message, d.build_duration_seconds
		FROM deployments d
		JOIN projects p ON p.id = d.project_id
		WHERE d.id = $1 AND p.user_id = $2
		LIMIT 1
	`, id, userID).Scan(
		&resp.ID,
		&projectID,
		&resp.Branch,
		&resp.Status,
		&resp.AttempCount,
		&resp.MaxAttempts,
		&resp.CreatedAt,
		&startedAt,
		&finishedAt,
		&errorMessage,
		&buildDurationSeconds,
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

	if projectID.Valid {
		resp.ProjectID = &projectID.String
	}
	resp.StartedAt = nullTimeToStringPtr(startedAt)
	resp.FinishedAt = nullTimeToStringPtr(finishedAt)
	resp.ErrorMessage = nullStringToPtr(errorMessage)
	resp.BuildDurationSeconds = nullInt64ToIntPtr(buildDurationSeconds)
	resp.URL = "http://localhost:8001/" + resp.ID

	return c.JSON(http.StatusOK, resp)
}

func GetDeployments(c *echo.Context) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	limit := 20
	offset := 0

	if raw := c.QueryParam("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 100 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "limit must be an integer between 1 and 100",
			})
		}
		limit = parsed
	}

	if raw := c.QueryParam("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "offset must be an integer greater than or equal to 0",
			})
		}
		offset = parsed
	}

	rows, err := db.DB.Query(`
		SELECT d.id, d.project_id, d.branch, d.status, d.attempt_count, d.max_attempts, d.created_at, d.started_at, d.finished_at, d.error_message, d.build_duration_seconds
		FROM deployments d
		JOIN projects p ON p.id = d.project_id
		WHERE p.user_id = $1
		ORDER BY d.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch deployments",
		})
	}
	defer rows.Close()

	deployments := make([]DeploymentResponse, 0, limit)
	for rows.Next() {
		var (
			projectID    sql.NullString
			id           string
			branch       string
			status       string
			attemptCount int
			maxAttempts  int
			createdAt    time.Time
			startedAt    sql.NullTime
			finishedAt   sql.NullTime
			errorMessage sql.NullString
			buildSeconds sql.NullInt64
		)

		if err := rows.Scan(&id, &projectID, &branch, &status, &attemptCount, &maxAttempts, &createdAt, &startedAt, &finishedAt, &errorMessage, &buildSeconds); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan deployment row",
			})
		}

		var projectIDPtr *string
		if projectID.Valid {
			projectIDPtr = &projectID.String
		}

		deployments = append(deployments, DeploymentResponse{
			ID:                   id,
			ProjectID:            projectIDPtr,
			Branch:               branch,
			Status:               status,
			AttempCount:          fmt.Sprintf("%d", attemptCount),
			MaxAttempts:          fmt.Sprintf("%d", maxAttempts),
			CreatedAt:            createdAt.Format(time.RFC3339),
			StartedAt:            nullTimeToStringPtr(startedAt),
			FinishedAt:           nullTimeToStringPtr(finishedAt),
			ErrorMessage:         nullStringToPtr(errorMessage),
			BuildDurationSeconds: nullInt64ToIntPtr(buildSeconds),
			URL:                  "http://localhost:8001/" + id,
		})
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to iterate deployments",
		})
	}

	return c.JSON(http.StatusOK, deployments)
}

func nullTimeToStringPtr(value sql.NullTime) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.Format(time.RFC3339)
	return &formatted
}

func nullStringToPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := value.String
	return &trimmed
}

func nullInt64ToIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
