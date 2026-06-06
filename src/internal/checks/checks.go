package checks

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/potibm/netprobe/src/internal/config"
	"github.com/potibm/netprobe/src/internal/domain"
	netprobe_net "github.com/potibm/netprobe/src/internal/net"
)

type CheckRunner struct {
	targets []domain.Target
	logger  *slog.Logger
	name    string
	cfg     config.Defaults
	clients domain.ClientList
}

func NewCheckRunner(
	cfg config.Config,
	iface string,
	family netprobe_net.IPFamily,
	logger *slog.Logger,
) *CheckRunner {
	runner := &CheckRunner{
		targets: cfg.BuildTargets(),
		clients: make(map[netprobe_net.IPFamily]*http.Client),
		logger:  logger.With("name", cfg.Name),
		name:    cfg.Name,
		cfg:     cfg.Defaults,
	}

	timeout := time.Duration(cfg.Defaults.TimeoutSeconds) * time.Second

	if family == netprobe_net.IP4 || family == netprobe_net.IPAuto {
		if client, err := createClientForInterface(iface, netprobe_net.IP4, timeout); err == nil {
			runner.clients[netprobe_net.IP4] = client
		} else {
			logger.Warn("⚠️ Could not create IPv4 client for interface", "interface", iface, "error", err)
		}
	}

	if family == netprobe_net.IP6 || family == netprobe_net.IPAuto {
		if client, err := createClientForInterface(iface, netprobe_net.IP6, timeout); err == nil {
			runner.clients[netprobe_net.IP6] = client
		} else {
			logger.Warn("⚠️ Could not create IPv6 client for interface", "interface", iface, "error", err)
		}
	}

	return runner
}

func (cr *CheckRunner) Run(ctx context.Context) {
	cr.logger.Info("🚀 Starting CheckRunner loop", "interval", cr.cfg.IntervalSeconds)

	interval := time.Duration(cr.cfg.IntervalSeconds) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	cr.runOnce()

	for {
		select {
		case <-ctx.Done():
			cr.logger.Info("🛑 Shutting down CheckRunner gracefully")

			return

		case <-ticker.C:
			cr.runOnce()
		}
	}
}

func (cr *CheckRunner) runOnce() {
	for _, target := range cr.targets {
		log := cr.logger.With("target", target.Hostname)

		log.Debug("Running checks for target", "target", target.Hostname)

		for _, check := range target.Checks {
			log.Debug("Running check", "check", check.Name())
			result := check.Execute(cr.clients, log)

			cr.handleResult(target, result, log)
		}
	}
}

func (cr *CheckRunner) handleResult(target domain.Target, result domain.CheckResult, log *slog.Logger) {
	if result.Success {
		log.Debug("✅ Check passed", "check", result.CheckName)

		return
	}

	log.Error("❌ Check failed",
		"check", result.CheckName,
		"error", result.ErrorMessage,
	)
}
