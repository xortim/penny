package cmd

import (
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xortim/penny/conf"
)

var cfgFile string

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Version: conf.GitVersion,
		Use:     conf.Executable,
		Short:   "Penny is a community moderation bot.",
		Long:    `Penny helps community admins do their job.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
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
	c.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default \"$HOME/."+conf.Executable+".yaml\")")
	c.MarkPersistentFlagFilename("config")

	c.PersistentFlags().StringSlice("global_admins", []string{}, "A string list of global admin UUIDs.")
	viper.BindPFlag("slack.global_admins", c.PersistentFlags().Lookup("global_admins"))

	c.PersistentFlags().String("slack_oauth_token", "", "Slack App OAuth token.")
	viper.BindPFlag("slack.oauth_token", c.PersistentFlags().Lookup("slack_oauth_token"))
	viper.BindEnv("slack.oauth_token", "SLACK_OAUTH_TOKEN")

	c.PersistentFlags().String("slack_signing_secret", "", "Slack secret used for message signing.")
	viper.BindPFlag("slack.signing_secret", c.PersistentFlags().Lookup("slack_signing_secret"))
	viper.BindEnv("slack.signing_secret", "SLACK_SIGNING_SECRET")

	c.PersistentFlags().String("spam_feed_channel", "spam-feed", "Slack channel where Racji App reports SPAM posts.")
	viper.BindPFlag("spam_feed.channel", c.PersistentFlags().Lookup("spam_feed.channel"))

	c.PersistentFlags().String("spam_feed_emoji", "no_good", "Slack emoji configured for Racji App to report SPAM posts.")
	viper.BindPFlag("spam_feed.emoji", c.PersistentFlags().Lookup("spam_feed.emoji"))

	c.PersistentFlags().String("spam_feed_reaction_emoji", "thumbsup", "The reaction Penny adds to the reported spam-feed post with.")
	viper.BindPFlag("spam_feed.reaction_emoji", c.PersistentFlags().Lookup("spam_feed.reaction_emoji"))

	c.PersistentFlags().String("spam_feed_response", "", "Threaded message response to the feed message. If empty, no thread is started.")
	viper.BindPFlag("spam_feed.response", c.PersistentFlags().Lookup("spam_feed.response"))
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
		viper.SetConfigName("." + conf.Executable)
	}

	viper.SetTypeByDefaultValue(true)
	viper.SetEnvPrefix(conf.Executable)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		println("Using config file: ", viper.ConfigFileUsed())
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}
