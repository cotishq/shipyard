package logs

import (
	"strings"
	"testing"
)

func TestTruncateLogMessage_LeavesShortMessagesUntouched(t *testing.T) {
	t.Setenv("MAX_LOG_SIZE_BYTES", "100")

	message := "short message"
	got := truncateLogMessage(message)
	if got != message {
		t.Fatalf("expected message to remain unchanged, got %q", got)
	}
}

func TestTruncateLogMessage_TruncatesLongMessages(t *testing.T) {
	t.Setenv("MAX_LOG_SIZE_BYTES", "20")

	got := truncateLogMessage(strings.Repeat("a", 50))
	if len(got) != 20 {
		t.Fatalf("expected truncated length 20, got %d", len(got))
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}
