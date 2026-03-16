package logs

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/cotishq/shipyard/internal/db"
)

func AddLog(deploymentID string, message string) {
	message = truncateLogMessage(message)

	_, err := db.DB.Exec(`
		INSERT INTO deployment_logs (deployment_id, message)
		VALUES ($1, $2)
	`, deploymentID, message)

	if err != nil {
		log.Println("failed to insert log:", err)
	}
}

func truncateLogMessage(message string) string {
	maxLen := getEnvInt("MAX_LOG_SIZE_BYTES", 8192)
	if maxLen <= 0 {
		return ""
	}
	if len(message) <= maxLen {
		return message
	}

	suffix := "\n...[truncated]"
	if maxLen <= len(suffix) {
		return suffix[:maxLen]
	}

	cutoff := maxLen - len(suffix)
	if cutoff < 0 {
		cutoff = 0
	}

	return message[:cutoff] + suffix
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
