package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "supportctl",
	Short: "Tooling for mattermost support",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.supportctl.yaml)")

	rootCmd.PersistentFlags().String("zendesk.subdomain", "", "Zendesk subdomain")
	viper.BindPFlag("zendesk.subdomain", rootCmd.PersistentFlags().Lookup("zendesk.subdomain"))

	rootCmd.PersistentFlags().String("zendesk.bearer-token", "", "Zendesk bearer token")
	viper.BindPFlag("zendesk.bearer-token", rootCmd.PersistentFlags().Lookup("zendesk.bearer-token"))

	rootCmd.PersistentFlags().String("work-dir", ".", "location of the work directory for the tickets")
	viper.BindPFlag("work-dir", rootCmd.PersistentFlags().Lookup("work-dir"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".supportctl")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			cobra.CheckErr(err)
		}
	}
}
