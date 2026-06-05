package cmd

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/potibm/netprobe/src/internal/checks"
	"github.com/potibm/netprobe/src/internal/config"
	netprobe_net "github.com/potibm/netprobe/src/internal/net"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultTimeout = 5 * time.Second
)

func NewCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "netprobe",
		Short:   "A network monitoring tool for the Evoke demoparty",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			logLevel := getSlogLevel(viper.GetString("log-level"))
			slog.SetLogLoggerLevel(logLevel)

			logger := slog.Default().With("app", "netprobe")

			logger.Info("🚀 Starting netprobe", "version", version)

			iface := viper.GetString("interface")
			configFile := viper.GetString("config")
			ipFamily := netprobe_net.IPFamily(viper.GetString("ip-family"))

			logger.Debug(
				"ℹ️ Options",
				"interface",
				iface,
				"config",
				configFile,
				"loglevel",
				logLevel,
				"ipfamily",
				ipFamily,
			)

			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				return fmt.Errorf("❌ Failed to load config: %w", err)
			}

			logger.Info("⚙️ Loaded config", "name", cfg.Name, "probes", len(cfg.Probes))

			client, err := httpClientForInterface(iface, ipFamily, defaultTimeout)
			if err != nil {
				return fmt.Errorf("❌ Failed to create HTTP client: %w", err)
			}

			checkRunner := checks.NewCheckRunner(cfg.Name, cfg.Defaults, cfg.BuildTargets(), client, logger)
			checkRunner.Run(ctx)

			return nil
		},
	}

	cmd.Flags().StringP("interface", "i", "", "Network interface to bind to (required)")
	cmd.Flags().StringP("config", "c", "", "Path to probe configuration file (required)")
	cmd.Flags().String("log-level", "info", "Logging level (debug, info, warn, error)")
	cmd.Flags().String("ip-family", "4", "IP family to use (4, 6, auto)")

	_ = viper.BindPFlags(cmd.Flags())

	viper.SetEnvPrefix("NETPROBE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	_ = cmd.MarkFlagRequired("interface")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}

func httpClientForInterface(iface string, family netprobe_net.IPFamily, timeout time.Duration) (*http.Client, error) {
	ip, err := netprobe_net.PickInterfaceIP(iface, family, false)
	if err != nil {
		return nil, err
	}

	slog.Info("ᯤ Using interface", "interface", iface, "ip", ip.String(), "ipfamily", family)

	dialer := &net.Dialer{
		Timeout: timeout,
		LocalAddr: &net.TCPAddr{
			IP: ip,
		},
	}

	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}
