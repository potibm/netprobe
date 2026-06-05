package domain

import (
	"log/slog"
	"net/http"
)

type WebsocketCheck struct {
	URL             string
	ExpectUp        bool
	ExpectValidCert bool // Spezifisch nur für HTTPS
}

func (c *WebsocketCheck) Name() string {
	return "WEBSOCKET"
}

func (c *WebsocketCheck) Execute(client *http.Client, log *slog.Logger) CheckResult {
	return CheckResult{CheckName: c.Name(), Success: true}
}
