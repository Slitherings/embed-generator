package main

import (
	"math/rand"
	"time"

	"github.com/merlinfuchs/embed-generator/embedg-server/api"
	"github.com/merlinfuchs/embed-generator/embedg-server/buildinfo"
	"github.com/merlinfuchs/embed-generator/embedg-server/config"
	"github.com/merlinfuchs/embed-generator/embedg-server/db/postgres/transfer"
	"github.com/merlinfuchs/embed-generator/embedg-server/migrate"
	"github.com/merlinfuchs/embed-generator/embedg-server/telemetry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:              "embedg",
	Short:            "The Friendly Service is a binary that serves as the (non-cloudflare) backend of Friendly Captcha's offering.",
	Long:             `The Friendly Service is a binary that serves as the (non-cloudflare) backend of Friendly Captcha's offering.`,
	PersistentPreRun: bindFlags,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "Config file (default is $HOME/.friendly.yaml)")
	rootCmd.Version = buildinfo.Version() + " " + buildinfo.Target() + " (" + buildinfo.CommitDate() + ") " + buildinfo.Commit()

	rootCmd.PersistentFlags().BoolP("debug", "D", false, "Debug mode (prints debug messages and call traces)")

	rootCmd.AddCommand(&cobra.Command{
		Use: "server",
		Run: func(cmd *cobra.Command, args []string) {
			api.Serve()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use: "transferdb",
		Run: func(cmd *cobra.Command, args []string) {
			transfer.TransferDB()
		},
	})
	rootCmd.AddCommand(migrate.Setup())
}

func bindFlags(cmd *cobra.Command, args []string) {
	viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
	viper.BindPFlag("cfg.local", cmd.Flags().Lookup("cfg.local"))
	viper.BindPFlag("cfg.local_file", cmd.Flags().Lookup("cfg.local_file"))
	viper.BindPFlag("cfg.remote", cmd.Flags().Lookup("cfg.remote"))
	viper.BindPFlag("cfg.remote_file", cmd.Flags().Lookup("cfg.remote_file"))
	viper.BindPFlag("cfg.watch", cmd.Flags().Lookup("cfg.watch"))
	viper.BindPFlag("cfg.watch_interval_sec", cmd.Flags().Lookup("cfg.watch_interval_sec"))
}

func main() {
	config.InitConfig()
	telemetry.SetupLogger()

	rand.Seed(time.Now().UnixNano())
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
