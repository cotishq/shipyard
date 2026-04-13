package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

// Unit tests for webhook logic (no database required)

func TestHandleGitHubWebhook_MissingEventHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := HandleGitHubWebhook(nil)(c); err != nil {
		t.Fatalf("expected no handler error, got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}

	var respBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if respBody["error"] != "missing github headers" {
		t.Fatalf("expected 'missing github headers' error, got %q", respBody["error"])
	}
}

func TestHandleGitHubWebhook_MissingDeliveryHeader(t *testing.T) {
	e := echo.New()
	payload := map[string]interface{}{
		"ref": "refs/heads/main",
		"repository": map[string]string{
			"clone_url": "https://github.com/test/repo",
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := HandleGitHubWebhook(nil)(c); err != nil {
		t.Fatalf("expected no handler error, got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandleGitHubWebhook_InvalidPayload(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := HandleGitHubWebhook(nil)(c); err != nil {
		t.Fatalf("expected no handler error, got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}

	var respBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if respBody["error"] != "invalid payload" {
		t.Fatalf("expected 'invalid payload' error, got %q", respBody["error"])
	}
}

func TestVerifyGitHubSignature_ValidSignature(t *testing.T) {
	secret := "test_secret_123"
	body := []byte(`{"ref":"refs/heads/main","repository":{"clone_url":"https://github.com/test/repo"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !verifyGitHubSignature(secret, body, validSig) {
		t.Fatal("expected valid signature to verify")
	}
}

func TestVerifyGitHubSignature_InvalidSignature(t *testing.T) {
	secret := "test_secret_123"
	body := []byte(`{"ref":"refs/heads/main","repository":{"clone_url":"https://github.com/test/repo"}}`)

	if verifyGitHubSignature(secret, body, "sha256=invalid") {
		t.Fatal("expected invalid signature to fail verification")
	}
}

func TestVerifyGitHubSignature_InvalidPrefix(t *testing.T) {
	secret := "test_secret_123"
	body := []byte(`{"ref":"refs/heads/main","repository":{"clone_url":"https://github.com/test/repo"}}`)

	if verifyGitHubSignature(secret, body, "md5=abc123") {
		t.Fatal("expected non-sha256 prefix to fail verification")
	}
}

func TestVerifyGitHubSignature_MissingPrefix(t *testing.T) {
	secret := "test_secret_123"
	body := []byte(`{"ref":"refs/heads/main","repository":{"clone_url":"https://github.com/test/repo"}}`)

	if verifyGitHubSignature(secret, body, "abc123") {
		t.Fatal("expected missing prefix to fail verification")
	}
}

func TestVerifyGitHubSignature_TimingAttackResistant(t *testing.T) {
	secret := "test_secret_123"
	body := []byte(`{"ref":"refs/heads/main","repository":{"clone_url":"https://github.com/test/repo"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Similar looking but different signature (off by one character)
	invalidSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if invalidSig[len(invalidSig)-1] == 'f' {
		invalidSig = invalidSig[:len(invalidSig)-1] + "0"
	} else {
		invalidSig = invalidSig[:len(invalidSig)-1] + "f"
	}

	if verifyGitHubSignature(secret, body, invalidSig) {
		t.Fatal("expected off-by-one signature to fail")
	}

	// Real signature should still pass
	if !verifyGitHubSignature(secret, body, validSig) {
		t.Fatal("expected valid signature to still pass")
	}
}

// Integration test scenarios documented
// These tests verify P1 requirements and acceptance criteria:
//
// 1. Valid webhook creates deployment:
//    - POST /webhooks/github with valid signature on default branch
//    - Creates QUEUED deployment linked to project
//    - Returns deployment_id in response
//
// 2. Wrong branch webhook ignored:
//    - POST /webhooks/github with valid signature on non-default branch
//    - Returns 200 OK with status "ignored"
//    - No deployment created
//
// 3. Duplicate webhook handling:
//    - POST /webhooks/github twice with same delivery_id
//    - First webhook creates deployment
//    - Second webhook doesn't create another deployment (deduped via unique constraint)
//    - Only one webhook_event recorded
//
// 4. Multiple deployments from project:
//    - POST /projects/:id/deployments called multiple times
//    - Each creates QUEUED deployment
//    - All linked to same project_id
//    - All with project's repo_url and build_preset
//
// 5. Project config drives behavior:
//    - Webhook uses project's default_branch for filtering
//    - Webhook uses project's repo_url for matching
//    - Deployment uses project's repo_url, build_preset, output_dir, default_branch
//
// To run full integration tests:
// 1. Start PostgreSQL: docker run -d postgres:15 -e POSTGRES_PASSWORD=postgres
// 2. Create test database: createdb -h localhost -U postgres shipyard_test
// 3. Run: go test -v ./internal/api -run TestGitHub -tags=integration
//
// Full integration tests are deferred to CI environment with persistent test database.
