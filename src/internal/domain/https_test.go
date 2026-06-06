package domain

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
	"github.com/stretchr/testify/assert"
)

func TestHTTPSFactsIsUp(t *testing.T) {
	facts := HTTPSFacts{
		netprobe_net.IP4: {IsUp: true},
		netprobe_net.IP6: {IsUp: false},
	}

	assert.True(t, facts.IsUp(netprobe_net.IP4))
	assert.False(t, facts.IsUp(netprobe_net.IP6))
	assert.False(t, facts.IsUp("missing"))
}

func TestHTTPSFactsAnyUp(t *testing.T) {
	empty := HTTPSFacts{}
	assert.False(t, empty.AnyUp())

	allDown := HTTPSFacts{
		netprobe_net.IP4: {IsUp: false},
	}
	assert.False(t, allDown.AnyUp())

	oneUp := HTTPSFacts{
		netprobe_net.IP4: {IsUp: false},
		netprobe_net.IP6: {IsUp: true},
	}
	assert.True(t, oneUp.AnyUp())
}

func TestHTTPSFactsStatusCode(t *testing.T) {
	facts := HTTPSFacts{
		netprobe_net.IP4: {StatusCode: 404},
	}

	assert.Equal(t, 404, facts.StatusCode(netprobe_net.IP4))
	assert.Equal(t, 0, facts.StatusCode(netprobe_net.IP6))
}

func TestHTTPSProtocolFactsHasExpectedCert(t *testing.T) {
	assert.False(t, (*HTTPSProtocolFacts)(nil).HasExpectedCert(true))

	valid := &HTTPSProtocolFacts{ValidCert: true}
	assert.True(t, valid.HasExpectedCert(true))
	assert.False(t, valid.HasExpectedCert(false))

	invalid := &HTTPSProtocolFacts{ValidCert: false}
	assert.False(t, invalid.HasExpectedCert(true))
	assert.True(t, invalid.HasExpectedCert(false))
}

func TestHTTPSCheckEvaluateIsolation(t *testing.T) {
	check := &HTTPSCheck{ExpectUp: false}

	// No leaks → success
	noLeaks := HTTPSFacts{
		netprobe_net.IP4: {IsUp: false},
	}
	result := check.evaluateIsolation(noLeaks)
	assert.True(t, result.Success)

	// Leaks detected → failure
	leaks := HTTPSFacts{
		netprobe_net.IP4: {IsUp: true},
		netprobe_net.IP6: {IsUp: true},
	}
	result = check.evaluateIsolation(leaks)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "SECURITY ALERT")
}

func TestHTTPSCheckEvaluateAvailability(t *testing.T) {
	check := &HTTPSCheck{ExpectUp: true, ExpectValidCert: true}

	// All down → failure
	allDown := HTTPSFacts{
		netprobe_net.IP4: {IsUp: false},
	}
	result := check.evaluateAvailability(allDown)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "offline")

	// Up but invalid cert (when valid expected) → failure
	badCert := HTTPSFacts{
		netprobe_net.IP4: {IsUp: true, ValidCert: false},
	}
	result = check.evaluateAvailability(badCert)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Certificate")

	// Up with valid cert → success
	correct := HTTPSFacts{
		netprobe_net.IP4: {IsUp: true, ValidCert: true},
		netprobe_net.IP6: {IsUp: true, ValidCert: true},
	}
	result = check.evaluateAvailability(correct)
	assert.True(t, result.Success)
}

func TestHTTPSCheckEvaluateAvailabilityExpectInvalidCert(t *testing.T) {
	check := &HTTPSCheck{ExpectUp: true, ExpectValidCert: false}

	// Up with valid cert (when invalid expected) → failure
	badCert := HTTPSFacts{
		netprobe_net.IP4: {IsUp: true, ValidCert: true},
	}
	result := check.evaluateAvailability(badCert)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "valid")
}

func TestHTTPSCheckEvaluateNoClients(t *testing.T) {
	check := &HTTPSCheck{ExpectUp: true}
	result := check.evaluate(HTTPSFacts{})
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "No active network clients")
}

func TestHTTPSCheckEvaluateRoutesToIsolation(t *testing.T) {
	// When ExpectUp is false, evaluate() should route to evaluateIsolation
	check := &HTTPSCheck{ExpectUp: false}

	noLeaks := HTTPSFacts{
		netprobe_net.IP4: {IsUp: false},
	}
	result := check.evaluate(noLeaks)
	assert.True(t, result.Success)

	leaks := HTTPSFacts{
		netprobe_net.IP4: {IsUp: true},
	}
	result = check.evaluate(leaks)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "SECURITY ALERT")
}

func TestHTTPSCheckGatherSingleFactsSuccess(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/"}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
	client := newMockClient(resp, nil)

	facts := check.gatherSingleFacts(client)
	assert.NoError(t, facts.Error)
	assert.True(t, facts.IsUp)
	assert.True(t, facts.ValidCert)
	assert.Equal(t, http.StatusOK, facts.StatusCode)
}

func TestHTTPSCheckGatherSingleFactsBadStatus(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/"}
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("Not Found")),
	}
	client := newMockClient(resp, nil)

	facts := check.gatherSingleFacts(client)
	assert.NoError(t, facts.Error)
	assert.False(t, facts.IsUp)
	assert.True(t, facts.ValidCert)
	assert.Equal(t, http.StatusNotFound, facts.StatusCode)
}

func TestHTTPSCheckGatherSingleFactsCertError(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/"}
	client := newMockClient(nil, fmt.Errorf("x509: certificate signed by unknown authority"))

	facts := check.gatherSingleFacts(client)
	assert.Error(t, facts.Error)
	assert.True(t, facts.IsUp)
	assert.False(t, facts.ValidCert)
}

func TestHTTPSCheckGatherSingleFactsTLSError(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/"}
	client := newMockClient(nil, fmt.Errorf("tls: bad certificate"))

	facts := check.gatherSingleFacts(client)
	assert.Error(t, facts.Error)
	assert.True(t, facts.IsUp)
	assert.False(t, facts.ValidCert)
}

func TestHTTPSCheckGatherSingleFactsNetworkError(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/"}
	client := newMockClient(nil, fmt.Errorf("connection refused"))

	facts := check.gatherSingleFacts(client)
	assert.Error(t, facts.Error)
	assert.False(t, facts.IsUp)
	assert.False(t, facts.ValidCert)
}

func TestHTTPSCheckGatherFacts(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/"}

	respOK := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
	resp404 := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("Not Found")),
	}

	clients := ClientList{
		netprobe_net.IP4: newMockClient(respOK, nil),
		netprobe_net.IP6: newMockClient(resp404, nil),
	}

	facts := check.gatherFacts(clients)
	assert.Len(t, facts, 2)
	assert.True(t, facts[netprobe_net.IP4].IsUp)
	assert.False(t, facts[netprobe_net.IP6].IsUp)
}

func TestHTTPSCheckExecuteSuccess(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/", ExpectUp: true, ExpectValidCert: true}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp, nil),
	}

	result := check.Execute(clients, newDiscardLogger())
	assert.True(t, result.Success)
	assert.Equal(t, "HTTPS", result.CheckName)
}

func TestHTTPSCheckExecuteFailure(t *testing.T) {
	check := &HTTPSCheck{URL: "https://example.com/", ExpectUp: true, ExpectValidCert: true}

	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("Not Found")),
	}
	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp, nil),
	}

	result := check.Execute(clients, newDiscardLogger())
	assert.False(t, result.Success)
}

func TestHTTPSCheckExecuteIsolationSuccess(t *testing.T) {
	// Service is down → isolation confirmed
	check := &HTTPSCheck{URL: "https://example.com/", ExpectUp: false}

	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("Not Found")),
	}
	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp, nil),
	}

	result := check.Execute(clients, newDiscardLogger())
	assert.True(t, result.Success)
	assert.Equal(t, "HTTPS", result.CheckName)
}

func TestHTTPSCheckExecuteIsolationFailure(t *testing.T) {
	// Service is up when it should be isolated → security alert
	check := &HTTPSCheck{URL: "https://example.com/", ExpectUp: false}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
	clients := ClientList{
		netprobe_net.IP4: newMockClient(resp, nil),
	}

	result := check.Execute(clients, newDiscardLogger())
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "SECURITY ALERT")
}
