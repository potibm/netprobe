package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
)

type WebsocketCheck struct {
	URL string
}

type WebsocketProtocolFacts struct {
	Error               error
	StatusCode          int
	IsUp                bool
	HandshakeSuccessful bool
}

type WebsocketFacts map[netprobe_net.IPFamily]*WebsocketProtocolFacts

func (facts WebsocketFacts) IsUp(family netprobe_net.IPFamily) bool {
	f := facts[family]

	return f != nil && f.IsUp
}

func (facts WebsocketFacts) AnyUp() bool {
	for family := range facts {
		if facts.IsUp(family) {
			return true
		}
	}

	return false
}

func (c *WebsocketCheck) Name() string {
	return "WEBSOCKET"
}

func (c *WebsocketCheck) Execute(clients ClientList, log *slog.Logger) CheckResult {
	facts := c.gatherFacts(clients)

	result := c.evaluate(facts)

	if result.Success {
		log.Debug("✅ WEBSOCKET probe matches expectations", "url", c.URL)
	} else {
		log.Warn("⚠️ WEBSOCKET probe failed", "url", c.URL, "error", result.ErrorMessage)
	}

	return result
}

func (c *WebsocketCheck) gatherFacts(clients ClientList) WebsocketFacts {
	facts := make(WebsocketFacts)

	for family, client := range clients {
		facts[family] = c.gatherSingleFacts(client)
	}

	return facts
}

func (c *WebsocketCheck) gatherSingleFacts(client *http.Client) *WebsocketProtocolFacts {
	pFacts := &WebsocketProtocolFacts{}

	// Make the scheme compatible with the standard HTTP client (ws -> http)
	httpURL := c.URL
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)
	httpURL = strings.Replace(httpURL, "wss://", "https://", 1)

	req, err := http.NewRequest(http.MethodGet, httpURL, http.NoBody)
	if err != nil {
		pFacts.Error = err

		return pFacts
	}

	// Set the magic headers for the websocket upgrade
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")

	// A random key is required by RFC
	const keyLength = 16

	nonce := make([]byte, keyLength)
	_, _ = rand.Read(nonce)
	req.Header.Set("Sec-WebSocket-Key", base64.StdEncoding.EncodeToString(nonce))

	resp, err := client.Do(req)
	if err != nil {
		pFacts.Error = err
		pFacts.IsUp = false

		return pFacts
	}
	defer resp.Body.Close()

	pFacts.StatusCode = resp.StatusCode

	// The port is responding, so the service is basically reachable (IsUp)
	pFacts.IsUp = true

	// RFC 6455: The server MUST respond with 101 if the upgrade succeeds
	pFacts.HandshakeSuccessful = resp.StatusCode == http.StatusSwitchingProtocols

	return pFacts
}

func (c *WebsocketCheck) evaluate(facts WebsocketFacts) CheckResult {
	if len(facts) == 0 {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: "No active network clients available for the WebSocket check",
		}
	}

	// 1. OR logic: At least one IP channel must be able to reach the service
	if !facts.AnyUp() {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: "WebSocket service is completely offline (not reachable via IPv4 or IPv6)",
		}
	}

	// 2. Every channel through which the server responds MUST complete the upgrade successfully
	for family, f := range facts {
		if facts.IsUp(family) && !f.HandshakeSuccessful {
			return CheckResult{
				CheckName: c.Name(),
				Success:   false,
				ErrorMessage: fmt.Sprintf(
					"[%s] HTTP connection established, but protocol upgrade failed (Status: %d). Check proxy configuration!",
					family,
					f.StatusCode,
				),
			}
		}
	}

	return CheckResult{CheckName: c.Name(), Success: true}
}
