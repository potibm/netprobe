package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderError(t *testing.T) {
	assert.Contains(t, RenderError(ErrNotAvailable, ""), "offline")
	assert.Contains(t, RenderError(ErrNotAvailable, "IP4"), "[IP4]")

	assert.Contains(t, RenderError(ErrUnexpectedUp, "", []string{"IP4", "IP6"}), "SECURITY ALERT")

	assert.Contains(t, RenderError(ErrUnexpectedDown, "IP4", 503), "503")

	assert.Contains(t, RenderError(ErrWrongRedirect, "IP4", true, 301), "HTTPS redirect")
	assert.Contains(t, RenderError(ErrWrongRedirect, "IP4", false, 200), "NO HTTPS redirect")

	assert.Contains(t, RenderError(ErrInvalidCertificate, "IP4", "valid"), "valid")

	assert.Contains(t, RenderError(ErrWebsocketUpgradeFail, "IP4", 403), "protocol upgrade failed")
	assert.Contains(t, RenderError(ErrWebsocketUpgradeFail, "IP4", 403), "403")

	assert.Contains(t, RenderError(ErrNoClients, ""), "No active network clients")

	assert.Contains(t, RenderError("UNKNOWN_CODE", "IP4"), "Unknown network error")
}

func TestNewCheckResultFailure(t *testing.T) {
	result := NewCheckResultFailure("HTTP", ErrNotAvailable, "")
	assert.Equal(t, "HTTP", result.CheckName)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.ErrorMessage)
}

func TestNewCheckResultSuccess(t *testing.T) {
	result := NewCheckResultSuccess("HTTPS")
	assert.Equal(t, "HTTPS", result.CheckName)
	assert.True(t, result.Success)
	assert.Empty(t, result.ErrorMessage)
}
