package domain

import (
	"testing"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
	"github.com/stretchr/testify/assert"
)

func TestHTTPFactsIsUp(t *testing.T) {
	facts := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: true},
		netprobe_net.IP6: {Error: assert.AnError, IsUp: true},
	}

	assert.True(t, facts.IsUp(netprobe_net.IP4))
	assert.False(t, facts.IsUp(netprobe_net.IP6)) // has error
	assert.False(t, facts.IsUp("unknown"))        // missing
}

func TestHTTPFactsAnyUp(t *testing.T) {
	empty := HTTPFacts{}
	assert.False(t, empty.AnyUp())

	allDown := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: false},
	}
	assert.False(t, allDown.AnyUp())

	oneUp := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: false},
		netprobe_net.IP6: {Error: nil, IsUp: true},
	}
	assert.True(t, oneUp.AnyUp())
}

func TestHTTPFactsHasCorrectRedirect(t *testing.T) {
	facts := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: true, RedirectScheme: "https"},
		netprobe_net.IP6: {Error: nil, IsUp: true, RedirectScheme: "http"},
	}

	assert.True(t, facts.HasCorrectRedirect(netprobe_net.IP4, true))
	assert.False(t, facts.HasCorrectRedirect(netprobe_net.IP6, true))
	assert.False(t, facts.HasCorrectRedirect("missing", true))
}

func TestHTTPFactsStatusCode(t *testing.T) {
	facts := HTTPFacts{
		netprobe_net.IP4: {StatusCode: 200},
	}

	assert.Equal(t, 200, facts.StatusCode(netprobe_net.IP4))
	assert.Equal(t, 0, facts.StatusCode(netprobe_net.IP6))
}

func TestHTTPProtocolFactsHasCorrectRedirect(t *testing.T) {
	assert.False(t, (*HTTPProtocolFacts)(nil).HasCorrectRedirect(true))

	withError := &HTTPProtocolFacts{Error: assert.AnError, IsUp: true, RedirectScheme: "https"}
	assert.False(t, withError.HasCorrectRedirect(true))

	down := &HTTPProtocolFacts{Error: nil, IsUp: false, RedirectScheme: "https"}
	assert.False(t, down.HasCorrectRedirect(true))

	redirectHTTPS := &HTTPProtocolFacts{Error: nil, IsUp: true, RedirectScheme: "https"}
	assert.True(t, redirectHTTPS.HasCorrectRedirect(true))
	assert.False(t, redirectHTTPS.HasCorrectRedirect(false))

	redirectHTTP := &HTTPProtocolFacts{Error: nil, IsUp: true, RedirectScheme: "http"}
	assert.False(t, redirectHTTP.HasCorrectRedirect(true))
	assert.True(t, redirectHTTP.HasCorrectRedirect(false))
}

func TestHTTPCheckEvaluateIsolation(t *testing.T) {
	check := &HTTPCheck{ExpectUp: false}

	// No leaks → success
	noLeaks := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: false},
	}
	result := check.evaluateIsolation(noLeaks)
	assert.True(t, result.Success)
	assert.Equal(t, "HTTP", result.CheckName)

	// Leaks detected → failure
	leaks := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: true},
		netprobe_net.IP6: {Error: nil, IsUp: true},
	}
	result = check.evaluateIsolation(leaks)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "SECURITY ALERT")
}

func TestHTTPCheckEvaluateAvailability(t *testing.T) {
	check := &HTTPCheck{ExpectUp: true, ExpectRedirectToHTTPS: true}

	// All down → failure
	allDown := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: false},
	}
	result := check.evaluateAvailability(allDown)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "offline")

	// Up but wrong redirect → failure
	wrongRedirect := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: true, RedirectScheme: "http"},
	}
	result = check.evaluateAvailability(wrongRedirect)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "redirect")

	// Up with correct redirect → success
	correct := HTTPFacts{
		netprobe_net.IP4: {Error: nil, IsUp: true, RedirectScheme: "https"},
		netprobe_net.IP6: {Error: nil, IsUp: true, RedirectScheme: "https"},
	}
	result = check.evaluateAvailability(correct)
	assert.True(t, result.Success)
}

func TestHTTPCheckEvaluateNoClients(t *testing.T) {
	check := &HTTPCheck{ExpectUp: true}
	result := check.evaluate(HTTPFacts{})
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "No active network clients")
}
