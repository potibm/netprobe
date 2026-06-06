package domain

import (
	"io"
	"net/http"
	"strings"
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

func TestWebsocketCheckGatherSingleFactsHandshakeSuccess(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}
	resp := &http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	client := newMockClient(resp, nil)

	facts := check.gatherSingleFacts(client)
	assert.NoError(t, facts.Error)
	assert.True(t, facts.IsUp)
	assert.True(t, facts.HandshakeSuccessful)
	assert.Equal(t, http.StatusSwitchingProtocols, facts.StatusCode)
}

func TestWebsocketCheckGatherSingleFactsHandshakeRejected(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("Forbidden")),
	}
	client := newMockClient(resp, nil)

	facts := check.gatherSingleFacts(client)
	assert.NoError(t, facts.Error)
	assert.True(t, facts.IsUp)
	assert.False(t, facts.HandshakeSuccessful)
	assert.Equal(t, http.StatusForbidden, facts.StatusCode)
}

func TestWebsocketCheckGatherSingleFactsNetworkError(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}
	client := newMockClient(nil, assert.AnError)

	facts := check.gatherSingleFacts(client)
	assert.Error(t, facts.Error)
	assert.False(t, facts.IsUp)
}

func TestWebsocketCheckGatherFacts(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}

	resp101 := &http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	resp403 := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("Forbidden")),
	}

	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp101, nil),
		netprobe_net.IP6: newMockClient(resp403, nil),
	}

	facts := check.gatherFacts(clients)
	assert.Len(t, facts, 2)
	assert.True(t, facts[netprobe_net.IP4].IsUp)
	assert.True(t, facts[netprobe_net.IP4].HandshakeSuccessful)
	assert.True(t, facts[netprobe_net.IP6].IsUp)
	assert.False(t, facts[netprobe_net.IP6].HandshakeSuccessful)
}

func TestWebsocketCheckExecuteSuccess(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}

	resp := &http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp, nil),
	}

	result := check.Execute(clients, newDiscardLogger())
	assert.True(t, result.Success)
	assert.Equal(t, "WEBSOCKET", result.CheckName)
}

func TestWebsocketCheckExecuteFailure(t *testing.T) {
	check := &WebsocketCheck{URL: "wss://example.com/ws"}

	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("Forbidden")),
	}
	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp, nil),
	}

	result := check.Execute(clients, newDiscardLogger())
	assert.False(t, result.Success)
}
