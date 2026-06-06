package config

import (
	"github.com/potibm/netprobe/src/internal/domain"
)

func (c *Config) BuildTargets() []domain.Target {
	var targets []domain.Target

	for _, probe := range c.Probes {
		target := domain.Target{
			ID:       probe.ID,
			Hostname: probe.Hostname,
			Checks:   make([]domain.Check, 0),
		}

		if probe.HTTP != nil {
			target.Checks = append(target.Checks, &domain.HTTPCheck{
				URL:                   probe.HTTP.GetURL(probe.Hostname),
				ExpectUp:              probe.HTTP.ExpectUp,
				ExpectRedirectToHTTPS: probe.HTTP.DoesExpectRedirectToHTTP(),
			})
		}

		if probe.HTTPS != nil {
			target.Checks = append(target.Checks, &domain.HTTPSCheck{
				URL:             probe.HTTPS.GetURL(probe.Hostname),
				ExpectUp:        probe.HTTPS.ExpectUp,
				ExpectValidCert: probe.HTTPS.DoesExpectValidCert(),
			})
		}

		if probe.Websocket != nil {
			target.Checks = append(target.Checks, &domain.WebsocketCheck{
				URL: probe.Websocket.GetURL(probe.Hostname),
			})
		}

		targets = append(targets, target)
	}

	return targets
}
