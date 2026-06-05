package domain

import (
	"log/slog"
	"net/http"
)

type HTTPSCheck struct {
	URL             string
	ExpectUp        bool
	ExpectValidCert bool // Spezifisch nur für HTTPS
}

type HTTPSFacts struct {
	Error          error
	StatusCode     int
	IsUp           bool
	ValidCert      bool
}

func (c *HTTPSCheck) Name() string {
	return "HTTPS"
}

func (c *HTTPSCheck) Execute(client *http.Client, log *slog.Logger) CheckResult {

	facts := c.gatherFacts(client)

	log.Debug(
		"🔍 Gathered HTTPS facts",
		"url",
		c.URL,
		"status_code",
		facts.StatusCode,
		"is_up",
		facts.IsUp,
		"valid_cert",
		facts.ValidCert,
		"error",
		facts.Error,
	)

	return CheckResult{CheckName: c.Name(), Success: true}
}

func (c *HTTPSCheck) gatherFacts(client *http.Client) HTTPSFacts {
	facts := HTTPSFacts{}

	resp, err := client.Get(c.URL)
	if err != nil {
		facts.Error = err

		return facts
	}
	defer resp.Body.Close()

	facts.StatusCode = resp.StatusCode
	facts.IsUp = resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest

	return facts
}