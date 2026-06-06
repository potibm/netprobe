package domain

import (
	"testing"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketFactsIsUp(t *testing.T) {
	facts := WebsocketFacts{
		netprobe_net.IP4: {IsUp: true},
		netprobe_net.IP6: {IsUp: false},
	}

	assert.True(t, facts.IsUp(netprobe_net.IP4))
	assert.False(t, facts.IsUp(netprobe_net.IP6))
	assert.False(t, facts.IsUp("missing"))
}

func TestWebsocketFactsAnyUp(t *testing.T) {
	empty := WebsocketFacts{}
	assert.False(t, empty.AnyUp())

	allDown := WebsocketFacts{
		netprobe_net.IP4: {IsUp: false},
	}
	assert.False(t, allDown.AnyUp())

	oneUp := WebsocketFacts{
		netprobe_net.IP4: {IsUp: false},
		netprobe_net.IP6: {IsUp: true},
	}
	assert.True(t, oneUp.AnyUp())
}

func TestWebsocketCheckEvaluate(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}

	// No clients → failure
	noClients := WebsocketFacts{}
	result := check.evaluate(noClients)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "No active network clients")

	// All down → failure
	allDown := WebsocketFacts{
		netprobe_net.IP4: {IsUp: false},
	}
	result = check.evaluate(allDown)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "offline")

	// Up but handshake failed → failure
	handshakeFailed := WebsocketFacts{
		netprobe_net.IP4: {IsUp: true, HandshakeSuccessful: false, StatusCode: 403},
	}
	result = check.evaluate(handshakeFailed)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "protocol upgrade failed")
	assert.Contains(t, result.ErrorMessage, "403")

	// Up and handshake successful → success
	allGood := WebsocketFacts{
		netprobe_net.IP4: {IsUp: true, HandshakeSuccessful: true},
		netprobe_net.IP6: {IsUp: true, HandshakeSuccessful: true},
	}
	result = check.evaluate(allGood)
	assert.True(t, result.Success)
}
