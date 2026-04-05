package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

func APIKeyAuthMiddleware(db *sql.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token := strings.TrimSpace(c.Request().Header.Get("X-API-Key"))
			if token == "" {
				auth := strings.TrimSpace(c.Request().Header.Get("Authorization"))
				const bearer = "Bearer "
				if strings.HasPrefix(auth, bearer) {
					token = strings.TrimSpace(strings.TrimPrefix(auth, bearer))
				}
			}

			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			tokenHash := hashToken(token)

			var userID string
			err := db.QueryRow(`
				SELECT user_id
				FROM api_tokens
				WHERE token_hash = $1
				  AND is_active = TRUE
				  AND (expires_at IS NULL OR expires_at > NOW())
				LIMIT 1
			`, tokenHash).Scan(&userID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			c.Set("user_id", userID)

			return next(c)
		}
	}
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

type InMemoryRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateEntry
}

func NewInMemoryRateLimiter(limit int, window time.Duration) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateEntry),
	}
}

func (l *InMemoryRateLimiter) Allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.After(entry.resetAt) {
		l.entries[key] = rateEntry{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true, l.limit - 1
	}

	if entry.count >= l.limit {
		return false, 0
	}

	entry.count++
	l.entries[key] = entry
	return true, l.limit - entry.count
}

func RateLimitMiddleware(limiter *InMemoryRateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			key := strings.TrimSpace(c.RealIP())
			if key == "" {
				key = "unknown"
			}

			allowed, remaining := limiter.Allow(key, time.Now().UTC())
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			if !allowed {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "rate limit exceeded",
				})
			}

			return next(c)
		}
	}
}
