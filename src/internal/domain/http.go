package domain

import (
	"fmt"
	"log/slog"
	"net/http"
)

type HTTPCheck struct {
	URL                   string
	ExpectUp              bool
	ExpectRedirectToHTTPS bool
}

type HTTPFacts struct {
	Error          error
	StatusCode     int
	IsUp           bool
	RedirectScheme string
}

func (c *HTTPCheck) Name() string {
	return "HTTP"
}

func (c *HTTPCheck) Execute(client *http.Client, log *slog.Logger) CheckResult {
	facts := c.gatherFacts(client)
	log.Debug(
		"🔍 Gathered HTTP facts",
		"url",
		c.URL,
		"status_code",
		facts.StatusCode,
		"is_up",
		facts.IsUp,
		"redirect_scheme",
		facts.RedirectScheme,
		"error",
		facts.Error,
	)

	result := c.evaluate(facts)

	if result.Success {
		log.Debug("✅ HTTP probe matches expectations", "url", c.URL)
	} else {
		log.Warn("⚠️ HTTP probe failed", "url", c.URL, "error", result.ErrorMessage)
	}

	return result
}

func (c *HTTPCheck) gatherFacts(client *http.Client) HTTPFacts {
	facts := HTTPFacts{}

	resp, err := client.Get(c.URL)
	if err != nil {
		facts.Error = err

		return facts
	}
	defer resp.Body.Close()

	facts.StatusCode = resp.StatusCode
	facts.IsUp = resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest

	if resp.StatusCode >= 300 && resp.StatusCode <= 308 {
		loc, err := resp.Location()
		if err == nil {
			facts.RedirectScheme = loc.Scheme
		}
	}

	return facts
}

func (c *HTTPCheck) evaluate(facts HTTPFacts) CheckResult {
	if facts.Error != nil {
		if !c.ExpectUp {
			return CheckResult{CheckName: c.Name(), Success: true}
		}

		return CheckResult{CheckName: c.Name(), Success: false, ErrorMessage: facts.Error.Error()}
	}

	if facts.IsUp != c.ExpectUp {
		msg := "Expected service to be DOWN, but it is UP"
		if c.ExpectUp {
			msg = "Expected service to be UP, but it is DOWN"
		}

		return CheckResult{
			CheckName:    c.Name(),
			Success:      false,
			ErrorMessage: fmt.Sprintf("%s (Status: %d)", msg, facts.StatusCode),
		}
	}

	if facts.IsUp && c.ExpectUp {
		isHTTPSRedirect := facts.RedirectScheme == "https"

		if isHTTPSRedirect != c.ExpectRedirectToHTTPS {
			msg := "Expected NO redirect to HTTPS, but got one"
			if c.ExpectRedirectToHTTPS {
				msg = "Expected redirect to HTTPS, but got none"
			}

			return CheckResult{CheckName: c.Name(), Success: false, ErrorMessage: msg}
		}
	}

	return CheckResult{CheckName: c.Name(), Success: true}
}
