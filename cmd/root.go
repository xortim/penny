package cmd

import (
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xortim/bones/conf"
)

var cfgFile string

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Version: conf.GitVersion,
		Use:     conf.Executable,
		Short:   "Bones is a bot for Slack's Events API",
		Long:    `Bones helps community admins do their job.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	cobra.OnInitialize(initConfig)
	rootCmd := newRootCmd()
	setupFlags(rootCmd)
	addSubcommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func setupFlags(c *cobra.Command) {
	c.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gadget.yaml)")
	c.MarkPersistentFlagFilename("config")

	c.PersistentFlags().StringSlice("global_admins", []string{}, "A string list of global admin UUIDs.")
	viper.BindPFlag("slack.global_admins", c.PersistentFlags().Lookup("global_admins"))

	c.PersistentFlags().String("slack_oauth_token", "", "Slack App OAuth token.")
	viper.BindPFlag("slack.oauth_token", c.PersistentFlags().Lookup("slack_oauth_token"))
	viper.BindEnv("slack.oauth_token", "SLACK_OAUTH_TOKEN")

	c.PersistentFlags().String("slack_signing_secret", "", "Slack secret used for message signing.")
	viper.BindPFlag("slack.signing_secret", c.PersistentFlags().Lookup("slack_signing_secret"))
	viper.BindEnv("slack.signing_secret", "SLACK_SIGNING_SECRET")
}

func addSubcommands(c *cobra.Command) {
	c.AddCommand(newVersionCmd())
	c.AddCommand(newServerCmd())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".gadget")
	}

	viper.SetTypeByDefaultValue(true)
	viper.SetEnvPrefix("GADGET")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		println("Using config file:", viper.ConfigFileUsed())
	}
}
