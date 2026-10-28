package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"atonixcorp/api/internal/security"
	"github.com/gin-gonic/gin"
)

func TestDecodeStrictJSONRejectsUnknownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test","unexpected":true}`))

	var body struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(ctx, &body, 1024); !errors.Is(err, security.ErrInvalid) {
		t.Fatalf("unknown JSON field should be rejected, got %v", err)
	}
}

func TestDecodeStrictJSONRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"too large"}`))

	var body struct {
		Name string `json:"name"`
	}
	err := decodeStrictJSON(ctx, &body, 8)
	if !errors.Is(err, security.ErrTooLarge) {
		t.Fatalf("oversized JSON body should be rejected, got %v", err)
	}
	writeSecurityError(ctx, err, "correlation-123")
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request returned %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(recorder.Body.String(), `"correlation_id":"correlation-123"`) {
		t.Fatalf("error response should preserve the correlation ID: %s", recorder.Body.String())
	}
}
