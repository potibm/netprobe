package cmd

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/potibm/netprobe/src/internal/checks"
	"github.com/potibm/netprobe/src/internal/config"
	"github.com/potibm/netprobe/src/internal/initializer"
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

			// Initialize telemetry
			shutdownFn, err := initializer.InitTelemetry(ctx, viper.GetString("otel-endpoint"), Version)
			if err != nil {
				return fmt.Errorf("failed to initialize telemetry: %w", err)
			}

			if shutdownFn != nil {
				defer shutdownFn()
			}

			logger := initializer.InitLogger(viper.GetString("log-format"), viper.GetString("log-level"))

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
				"ipfamily",
				ipFamily,
			)

			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				return fmt.Errorf("❌ Failed to load config: %w", err)
			}

			logger.Info("⚙️ Loaded config", "name", cfg.Name, "probes", len(cfg.Probes))

			checkRunner, err := checks.NewCheckRunner(*cfg, iface, ipFamily, logger)
			if err != nil {
				return fmt.Errorf("failed to create check runner: %w", err)
			}

			checkRunner.Run(ctx)

			return nil
		},
	}

	cmd.Flags().StringP("interface", "i", "", "Network interface to bind to (required)")
	cmd.Flags().StringP("config", "c", "", "Path to probe configuration file (required)")
	cmd.Flags().String("log-level", "info", "Logging level (debug, info, warn, error)")
	cmd.Flags().String("ip-family", "auto", "IP family to use (4, 6, auto)")
	cmd.Flags().String("log-format", "json", "Log format (json, text)")
	cmd.Flags().String("otel-endpoint", "", "OpenTelemetry exporter endpoint")

	_ = viper.BindPFlags(cmd.Flags())

	viper.SetEnvPrefix("NETPROBE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	_ = cmd.MarkFlagRequired("interface")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
