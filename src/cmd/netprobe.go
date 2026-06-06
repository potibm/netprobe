package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/joho/godotenv"
	"github.com/potibm/netprobe/src/internal/checks"
	"github.com/potibm/netprobe/src/internal/config"
	netprobe_net "github.com/potibm/netprobe/src/internal/net"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "netprobe",
		Short:   "A network monitoring tool for the Evoke demoparty",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			_ = godotenv.Load()

			logLevel := getSlogLevel(viper.GetString("log-level"))
			slog.SetLogLoggerLevel(logLevel)

			logger := slog.Default().With("app", "netprobe")

			logger.Info("🚀 Starting netprobe", "version", Version)

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

			checkRunner := checks.NewCheckRunner(*cfg, iface, ipFamily, logger)
			checkRunner.Run(ctx)

			return nil
		},
	}

	cmd.Flags().StringP("interface", "i", "", "Network interface to bind to (required)")
	cmd.Flags().StringP("config", "c", "", "Path to probe configuration file (required)")
	cmd.Flags().String("log-level", "info", "Logging level (debug, info, warn, error)")
	cmd.Flags().String("ip-family", "auto", "IP family to use (4, 6, auto)")

	_ = viper.BindPFlags(cmd.Flags())

	viper.SetEnvPrefix("NETPROBE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	_ = cmd.MarkFlagRequired("interface")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
