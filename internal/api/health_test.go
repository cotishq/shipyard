package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestGetHealth_OK(t *testing.T) {
	originalDBHealthCheck := dbHealthCheck
	originalStorageHealthCheck := storageHealthCheck
	t.Cleanup(func() {
		dbHealthCheck = originalDBHealthCheck
		storageHealthCheck = originalStorageHealthCheck
	})

	dbHealthCheck = func(context.Context) error { return nil }
	storageHealthCheck = func(context.Context) error { return nil }

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := GetHealth(c); err != nil {
		t.Fatalf("expected no handler error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestGetHealth_DegradedWhenDatabaseFails(t *testing.T) {
	originalDBHealthCheck := dbHealthCheck
	originalStorageHealthCheck := storageHealthCheck
	t.Cleanup(func() {
		dbHealthCheck = originalDBHealthCheck
		storageHealthCheck = originalStorageHealthCheck
	})

	dbHealthCheck = func(context.Context) error { return errors.New("db down") }
	storageHealthCheck = func(context.Context) error { return nil }

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := GetHealth(c); err != nil {
		t.Fatalf("expected no handler error, got %v", err)
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
	}
}
