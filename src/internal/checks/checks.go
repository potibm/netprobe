package checks

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/potibm/netprobe/src/internal/config"
	"github.com/potibm/netprobe/src/internal/domain"
)

type CheckRunner struct {
	targets []domain.Target
	client  *http.Client
	logger  *slog.Logger
	name    string
	cfg     config.Defaults
}

func NewCheckRunner(
	name string,
	cfg config.Defaults,
	targets []domain.Target,
	client *http.Client,
	logger *slog.Logger,
) *CheckRunner {
	return &CheckRunner{
		targets: targets,
		client:  client,
		logger:  logger.With("name", name),
		name:    name,
		cfg:     cfg,
	}
}

func (cr *CheckRunner) Run(ctx context.Context) {
	// @todo we need a ticker here to run this periodically, but for now we just run it once
	cr.runOnce()
}

func (cr *CheckRunner) runOnce() {
	for _, target := range cr.targets {
		log := cr.logger.With("target", target.Hostname)

		log.Debug("Running checks for target", "target", target.Hostname)

		for _, check := range target.Checks {
			log.Debug("Running check", "check", check.Name())
			result := check.Execute(cr.client, log)

			if !result.Success {
				log.Debug("Check failed", "check", result.CheckName, "error", result.ErrorMessage)
				// @TODO more actions beep (or similar)
			}
		}
	}
}
