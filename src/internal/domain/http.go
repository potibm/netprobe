package domain

import (
	"fmt"
	"log/slog"
	"net/http"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
)


type HTTPProtocolFacts struct {
	Error          error
	StatusCode     int
	IsUp           bool
	RedirectScheme string
}

func (f *HTTPProtocolFacts) HasCorrectRedirect(expected bool) bool {
	if f == nil || f.Error != nil || !f.IsUp {
		return false
	}
	actualRedirect := f.RedirectScheme == "https"
	return actualRedirect == expected
}

type HTTPCheck struct {
	URL                   string
	ExpectUp              bool
	ExpectRedirectToHTTPS bool
}

type HTTPFacts map[netprobe_net.IPFamily]*HTTPProtocolFacts

func (facts HTTPFacts) IsUp(family netprobe_net.IPFamily) bool {
	f := facts[family]
	return f != nil && f.Error == nil && f.IsUp
}

func (facts HTTPFacts) AnyUp() bool {
	for _, f := range facts {
		if f != nil && f.Error == nil && f.IsUp {
			return true
		}
	}
	return false
}

func (facts HTTPFacts) HasCorrectRedirect(family netprobe_net.IPFamily, expected bool) bool {
	f := facts[family]
	if f == nil || f.Error != nil || !f.IsUp {
		return false
	}
	actualRedirect := f.RedirectScheme == "https"
	return actualRedirect == expected
}

func (facts HTTPFacts) StatusCode(family netprobe_net.IPFamily) int {
	if f := facts[family]; f != nil {
		return f.StatusCode
	}
	return 0
}

func (c *HTTPCheck) Name() string {
	return "HTTP"
}

func (c *HTTPCheck) Execute(clients ClientList, log *slog.Logger) CheckResult {
	facts := c.gatherFacts(clients)

	result := c.evaluate(facts)

	if result.Success {
		log.Debug("✅ HTTP probe matches expectations", "url", c.URL)
	} else {
		log.Warn("⚠️ HTTP probe failed", "url", c.URL, "error", result.ErrorMessage)
	}

	return result
}

func (c *HTTPCheck) gatherFacts(clients ClientList) HTTPFacts {
	facts := make(HTTPFacts)

	for family, client := range clients {
		facts[family] = c.gatherSingleFacts(client)
	}

	return facts
}

func (c *HTTPCheck) gatherSingleFacts(client *http.Client) *HTTPProtocolFacts {
	pFacts := &HTTPProtocolFacts{}

	resp, err := client.Get(c.URL)
	if err != nil {
		pFacts.Error = err

		return pFacts
	}
	defer resp.Body.Close()

	pFacts.StatusCode = resp.StatusCode
	pFacts.IsUp = resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest

	if resp.StatusCode >= 300 && resp.StatusCode <= 308 {
		loc, err := resp.Location()
		if err == nil {
			pFacts.RedirectScheme = loc.Scheme
		}
	}

	return pFacts
}

func (c *HTTPCheck) evaluate(facts HTTPFacts) CheckResult {
	if len(facts) == 0 {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: "No active clients to perform the check",
		}
	}

	if c.ExpectUp {
		return c.evaluateAvailability(facts)
	}
	return c.evaluateIsolation(facts)
}

func (c *HTTPCheck) evaluateIsolation(facts HTTPFacts) CheckResult {
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
			ErrorMessage: fmt.Sprintf("🚨 SECURITY ALERT: Service should be isolated, but is responding over: %v", leaks),
		}
	}

	return CheckResult{CheckName: c.Name(), Success: true}
}

func (c *HTTPCheck) evaluateAvailability(facts HTTPFacts) CheckResult {
	// 1. One of the interfaces must be UP
	if !facts.AnyUp() {
		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: "Service is completely offline (not reachable via IPv4 or IPv6)",
		}
	}

	// 2. Any interface that is UP must have the correct redirect behavior
	for family, f := range facts {
		if facts.IsUp(family) && !f.HasCorrectRedirect(c.ExpectRedirectToHTTPS) {
			return CheckResult{
				CheckName:    c.Name(),
				Success:      false,
				ErrorMessage: fmt.Sprintf("[%s] Wrong redirect behavior (HTTP Status: %d)", family, f.StatusCode),
			}
		}
	}

	return CheckResult{CheckName: c.Name(), Success: true}
}