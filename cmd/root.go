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

	c.PersistentFlags().String("slack_user_oauth_token", "", "Slack App's User OAuth token.")
	viper.BindPFlag("slack.user_oauth_token", c.PersistentFlags().Lookup("slack_user_oauth_token"))
	viper.BindEnv("slack.user_oauth_token", "SLACK_USER_OAUTH_TOKEN")

	c.PersistentFlags().String("slack_oauth_token", "", "Slack App OAuth token.")
	viper.BindPFlag("slack.bot_oauth_token", c.PersistentFlags().Lookup("slack_oauth_token"))
	viper.BindEnv("slack.bot_oauth_token", "SLACK_OAUTH_TOKEN")

	c.PersistentFlags().String("slack_signing_secret", "", "Slack secret used for message signing.")
	viper.BindPFlag("slack.signing_secret", c.PersistentFlags().Lookup("slack_signing_secret"))
	viper.BindEnv("slack.signing_secret", "SLACK_SIGNING_SECRET")

	c.PersistentFlags().String("spam_feed_channel", "spam-feed", "Slack channel where Racji App reports SPAM posts.")
	viper.BindPFlag("spam_feed.channel", c.PersistentFlags().Lookup("spam_feed_channel"))

	c.PersistentFlags().String("spam_feed_emoji", "no_entry_sign", "Slack emoji configured for Racji App to report SPAM posts.")
	viper.BindPFlag("spam_feed.emoji", c.PersistentFlags().Lookup("spam_feed_emoji"))

	c.PersistentFlags().String("spam_feed_reaction_emoji_miss", "shrug", "The reaction Penny adds to the reported spam-feed post if below max_anomaly_score")
	viper.BindPFlag("spam_feed.reaction_emoji_miss", c.PersistentFlags().Lookup("spam_feed_reaction_emoji_miss"))

	c.PersistentFlags().String("spam_feed_reaction_emoji_hit", "no_good", "The reaction Penny adds to the reported spam-feed post if above max_anomaly_score.")
	viper.BindPFlag("spam_feed.reaction_emoji_hit", c.PersistentFlags().Lookup("spam_feed_reaction_emoji_hit"))

	c.PersistentFlags().String("reacji_response", "", "Threaded message response to the Reacji feed message. If empty, no thread is started.")
	viper.BindPFlag("spam_feed.reacji_response", c.PersistentFlags().Lookup("reacji_response"))

	c.PersistentFlags().String("op_warning", "This message has been flagged by our community as SPAM. The admins have been notified.", "Threaded message response to the OP as a warning when reported as SPAM. If empty, no thread is started.")
	viper.BindPFlag("spam_feed.op_warning", c.PersistentFlags().Lookup("op_warning"))

	c.PersistentFlags().String("local_timezone", "", "The local timezone of your community. This is the 'TZ Database' (Region/City_Name) format. Leave empty to not enforce this.")
	viper.BindPFlag("spam_feed.local_timezone", c.PersistentFlags().Lookup("local_timezone"))

	c.PersistentFlags().Int("activity_low_watermark", 10, "The minimum number of posts before adding to the user's anomaly score. Set this to 0 to disable.")
	viper.BindPFlag("spam_feed.activity_low_watermark", c.PersistentFlags().Lookup("activity_low_watermark"))

	c.PersistentFlags().Int("max_anomaly_score", 5, "The max anomaly score a post can reach before it is deleted.")
	viper.BindPFlag("spam_feed.max_anomaly_score", c.PersistentFlags().Lookup("max_anomaly_score"))

	c.PersistentFlags().Int("reported_score", 2, "The anomaly score to add to the post when it is reported.")
	viper.BindPFlag("spam_feed.anomaly_scores.reported", c.PersistentFlags().Lookup("reported_score"))

	c.PersistentFlags().Int("low_activity_score", 1, "The anomaly score to add to the reported post from users that are below the activity low watermark.")
	viper.BindPFlag("spam_feed.anomaly_scores.low_activity", c.PersistentFlags().Lookup("low_activity_score"))

	c.PersistentFlags().Int("outside_tz_score", 2, "The anomaly score to add to the reported post when the user is outside of the configured time zone.")
	viper.BindPFlag("spam_feed.anomaly_scores.outside_tz", c.PersistentFlags().Lookup("outside_tz_score"))
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
