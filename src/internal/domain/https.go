package domain

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
)

type HTTPSCheck struct {
	URL             string
	ExpectUp        bool
	ExpectValidCert bool 
}

type HTTPSProtocolFacts struct {
	Error          error
	StatusCode     int
	IsUp           bool
	ValidCert      bool
}

func (f *HTTPSProtocolFacts) HasExpectedCert(expected bool) bool {
	if f == nil {
		return false
	}
	return f.ValidCert == expected
}

type HTTPSFacts map[netprobe_net.IPFamily]*HTTPSProtocolFacts

func (facts HTTPSFacts) IsUp(family netprobe_net.IPFamily) bool {
	f := facts[family]
	return f != nil && f.IsUp
}

func (facts HTTPSFacts) AnyUp() bool {
	for family := range facts {
		if facts.IsUp(family) {
			return true
		}
	}
	return false
}

func (facts HTTPSFacts) StatusCode(family netprobe_net.IPFamily) int {
	if f := facts[family]; f != nil {
		return f.StatusCode
	}
	return 0
}

func (c *HTTPSCheck) Name() string {
	return "HTTPS"
}

func (c *HTTPSCheck) Execute(clients ClientList, log *slog.Logger) CheckResult {

	facts := c.gatherFacts(clients)

	result := c.evaluate(facts)

	if result.Success {
		log.Debug("✅ HTTP probe matches expectations", "url", c.URL)
	} else {
		log.Warn("⚠️ HTTP probe failed", "url", c.URL, "error", result.ErrorMessage)
	}

	return result
}

func (c *HTTPSCheck) gatherFacts(clients ClientList) HTTPSFacts {
	facts := make(HTTPSFacts)

	for family, client := range clients {
		facts[family] = c.gatherSingleFacts(client)
	}

	return facts
}

func (c *HTTPSCheck) gatherSingleFacts(client *http.Client) *HTTPSProtocolFacts {
	pFacts := &HTTPSProtocolFacts{}

	resp, err := client.Get(c.URL)
	
	if err != nil {
		pFacts.Error = err
		
		// Trick: Check whether the error was a certificate or TLS error
		if strings.Contains(err.Error(), "x509: certificate") || strings.Contains(err.Error(), "tls:") {
			pFacts.IsUp = true       // <-- IMPORTANT: Port 443 was open!
			pFacts.ValidCert = false // <-- But the certificate is invalid
		} else {
			// Real network error (e.g. no route to host, timeout)
			pFacts.IsUp = false 
		}
		return pFacts
	}
	defer resp.Body.Close()

	// If err == nil, Go has successfully validated the connection AND the certificate
	pFacts.ValidCert = true 
	pFacts.StatusCode = resp.StatusCode
	pFacts.IsUp = resp.StatusCode >= 200 && resp.StatusCode < 400

	return pFacts
}

func (c *HTTPSCheck) evaluate(facts HTTPSFacts) CheckResult {
	if len(facts) == 0 {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: "No active network clients available for the check",
		}
	}

	if c.ExpectUp {
		return c.evaluateAvailability(facts)
	}
	return c.evaluateIsolation(facts)
}

// Scenario 1: We expect isolation (firewall blocks)
func (c *HTTPSCheck) evaluateIsolation(facts HTTPSFacts) CheckResult {
	var leaks []string

	for family := range facts {
		if facts.IsUp(family) {
			leaks = append(leaks, string(family))
		}
	}

	if len(leaks) > 0 {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: fmt.Sprintf("🚨 SECURITY ALERT: Service should be isolated, but is responding on port 443 over: %v", leaks),
		}
	}

	return CheckResult{CheckName: c.Name(), Success: true}
}

// Scenario 2: We expect availability
func (c *HTTPSCheck) evaluateAvailability(facts HTTPSFacts) CheckResult {
	// 1. OR logic (at least one path must work)
	if !facts.AnyUp() {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: "Service is completely offline (not reachable via IPv4 or IPv6)",
		}
	}

	// 2. Validate certificate (but only on active channels)
	for family, f := range facts {
		if facts.IsUp(family) && !f.HasExpectedCert(c.ExpectValidCert) {
			msg := "invalid"
			if c.ExpectValidCert {
				msg = "valid"
			}
			return CheckResult{
				CheckName:    c.Name(),
				Success:      false,
				ErrorMessage: fmt.Sprintf("[%s] Certificate does not meet expectation (Expected: %s)", family, msg),
			}
		}
	}

	return CheckResult{CheckName: c.Name(), Success: true}
}