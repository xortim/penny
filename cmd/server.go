package cmd

import (
	"fmt"

	gadget "github.com/gadget-bot/gadget/core"
	helpers "github.com/gadget-bot/gadget/plugins/helpers"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xortim/penny/conf"
	"github.com/xortim/penny/gadgets/hallmonitor"
	"github.com/xortim/penny/gadgets/help"
	"github.com/xortim/penny/gadgets/whatsnew"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"serve"},
		Short:   "Run the bot",
		Long:    `Run the bot`,
		RunE:    server,
	}

	setupServerFlags(cmd)

	return cmd
}

func server(cmd *cobra.Command, args []string) error {
	myBot, err := gadget.SetupWithConfig(gadget.Config{
		SlackOAuthToken: viper.GetString("slack.bot_oauth_token"),
		SigningSecret:   viper.GetString("slack.signing_secret"),
		DBUser:          viper.GetString("db.username"),
		DBPass:          viper.GetString("db.password"),
		DBHost:          viper.GetString("db.hostname"),
		DBName:          viper.GetString("db.name"),
		ListenPort:      viper.GetString("server.port"),
		GlobalAdmins:    viper.GetStringSlice("slack.global_admins"),
	})
	if err != nil {
		return err
	}

	myBot.Router.AddChannelMessageRoutes(hallmonitor.GetChannelMessageRoutes())
	myBot.Router.AddMentionRoutes(whatsnew.GetMentionRoutes(ChangelogRaw))
	log.Debug().Int("changelog_bytes", len(ChangelogRaw)).Msg("registered what's new mention routes")

	myBot.Router.AddSlashCommandRoutes(help.GetSlashCommandRoutes())

	channelName := viper.GetString("spam_feed.channel")
	if channelName == "" {
		log.Debug().Msg("no spam-feed channel configured, skipping auto-join")
	} else if err := helpers.JoinChannelByName(*myBot.Client, channelName); err != nil {
		return fmt.Errorf("failed to join spam-feed channel: %w", err)
	} else {
		log.Info().Str("channel", channelName).Msg("joined spam-feed channel")
	}

	log.Info().
		Str("version", conf.GitVersion).
		Int("port", viper.GetInt("server.port")).
		Str("spam_feed_channel", viper.GetString("spam_feed.channel")).
		Msg("starting penny")

	return myBot.Run()
}

func setupServerFlags(c *cobra.Command) {
	c.PersistentFlags().IntP("port", "p", 3000, "The port on which the bot should bind.")
	_ = viper.BindPFlag("server.port", c.PersistentFlags().Lookup("port"))
	viper.RegisterAlias("listen.port", "server.port")
	viper.SetDefault("server.port", 3000)

	c.PersistentFlags().String("db_hostname", "localhost", "The host for "+conf.Executable+"'s DB.")
	_ = viper.BindPFlag("db.hostname", c.PersistentFlags().Lookup("db_hostname"))
	viper.RegisterAlias("db.host", "db.hostname")
	viper.SetDefault("db.hostname", "localhost")

	c.PersistentFlags().String("db_name", conf.Executable, "The name for "+conf.Executable+"'s DB.")
	_ = viper.BindPFlag("db.name", c.PersistentFlags().Lookup("db_name"))
	viper.SetDefault("db.name", conf.Executable)

	c.PersistentFlags().String("db_username", "", "The username for "+conf.Executable+"'s DB.")
	_ = viper.BindPFlag("db.username", c.PersistentFlags().Lookup("db_username"))
	viper.RegisterAlias("db.user", "db.username")
	viper.SetDefault("db.username", conf.Executable)

	c.PersistentFlags().String("db_password", "", "The password for "+conf.Executable+"'s DB.")
	_ = viper.BindPFlag("db.password", c.PersistentFlags().Lookup("db_password"))
	viper.RegisterAlias("db.pass", "db.password")
}
