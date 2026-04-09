package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type createTokenRequest struct {
	Label     string  `json:"label"`
	ExpiresIn *int64  `json:"expires_in_seconds,omitempty"`
}

type tokenResponse struct {
	ID         string  `json:"id"`
	Token      string  `json:"token,omitempty"`
	TokenPrefix string `json:"token_prefix"`
	Label      string  `json:"label"`
	IsActive   bool    `json:"is_active"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

func CreateToken(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		req := new(createTokenRequest)
		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		label := strings.TrimSpace(req.Label)
		if label == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "label is required"})
		}
		if len(label) > 80 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "label too long"})
		}

		raw, err := generateRawToken()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		}
		tokenHash := hashToken(raw)
		prefix := raw
		if len(prefix) > 12 {
			prefix = prefix[:12]
		}

		var expiresAt any = nil
		if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
			expiresAt = time.Now().UTC().Add(time.Duration(*req.ExpiresIn) * time.Second)
		}

		id := uuid.NewString()
		_, err = db.Exec(`
			INSERT INTO api_tokens (id, user_id, token_hash, token_prefix, label, is_active, expires_at)
			VALUES ($1, $2, $3, $4, $5, TRUE, $6)
		`, id, userID, tokenHash, prefix, label, expiresAt)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"id":    id,
			"token": raw, // return once
		})
	}
}

func ListTokens(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		rows, err := db.Query(`
			SELECT id, token_prefix, label, is_active, last_used_at, expires_at, created_at
			FROM api_tokens
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list tokens"})
		}
		defer rows.Close()

		out := make([]tokenResponse, 0)
		for rows.Next() {
			var (
				r tokenResponse
				lastUsed sql.NullTime
				expires  sql.NullTime
				created  time.Time
			)
			if err := rows.Scan(&r.ID, &r.TokenPrefix, &r.Label, &r.IsActive, &lastUsed, &expires, &created); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan token"})
			}
			if lastUsed.Valid {
				s := lastUsed.Time.Format(time.RFC3339)
				r.LastUsedAt = &s
			}
			if expires.Valid {
				s := expires.Time.Format(time.RFC3339)
				r.ExpiresAt = &s
			}
			r.CreatedAt = created.Format(time.RFC3339)
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to iterate tokens"})
		}

		return c.JSON(http.StatusOK, out)
	}
}

func RevokeToken(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID, err := authenticatedUserID(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "token id is required"})
		}

		res, err := db.Exec(`
			UPDATE api_tokens
			SET is_active = FALSE
			WHERE id = $1 AND user_id = $2
		`, id, userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to revoke token"})
		}

		affected, _ := res.RowsAffected()
		if affected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "token not found"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "revoked"})
	}
}

func generateRawToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "shipyard_" + hex.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}
