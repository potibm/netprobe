package initializer

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, logLevelFromString(tt.input))
		})
	}
}

func TestInitLogger_JSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := InitLogger("json", "debug")
	logger.Info("test message")

	w.Close()

	os.Stdout = old

	var buf bytes.Buffer

	_, _ = buf.ReadFrom(r)

	output := buf.String()

	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"level":"INFO"`)
}

func TestInitLogger_Text(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := InitLogger("text", "info")
	logger.Info("test message")

	w.Close()

	os.Stdout = old

	var buf bytes.Buffer

	_, _ = buf.ReadFrom(r)

	output := buf.String()

	assert.Contains(t, output, "msg=\"test message\"")
	assert.Contains(t, output, "level=INFO")
}

func TestInitLogger_DefaultsToJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := InitLogger("unknown", "info")
	logger.Info("test message")

	w.Close()

	os.Stdout = old

	var buf bytes.Buffer

	_, _ = buf.ReadFrom(r)

	output := buf.String()

	assert.Contains(t, output, `"msg":"test message"`)
}
