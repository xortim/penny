package cmd

import (
	"fmt"

	gadget "github.com/gadget-bot/gadget/core"
	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xortim/penny/conf"
	"github.com/xortim/penny/gadgets/hallmonitor"
	"github.com/xortim/penny/gadgets/whatsnew"
	"github.com/xortim/penny/pkg/slackclient"
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
	myBot, err := gadget.SetupWithConfig(
		viper.GetString("slack.bot_oauth_token"),
		viper.GetString("slack.signing_secret"),
		viper.GetString("db.username"),
		viper.GetString("db.password"),
		viper.GetString("db.hostname"),
		viper.GetString("db.name"),
		viper.GetString("db.port"),
		viper.GetStringSlice("slack.global_admins"))
	if err != nil {
		return err
	}

	myBot.Router.ChannelMessageRoutes = make(map[string]router.ChannelMessageRoute)
	myBot.Router.AddChannelMessageRoutes(hallmonitor.GetChannelMessageRoutes())

	myBot.Router.MentionRoutes = make(map[string]router.MentionRoute)
	myBot.Router.AddMentionRoutes(whatsnew.GetMentionRoutes(ChangelogRaw))
	log.Debug().Int("changelog_bytes", len(ChangelogRaw)).Msg("registered what's new mention routes")

	if err := joinSpamFeedChannel(myBot.Client); err != nil {
		return fmt.Errorf("failed to join spam-feed channel: %w", err)
	}

	log.Info().
		Str("version", conf.GitVersion).
		Int("port", viper.GetInt("server.port")).
		Str("spam_feed_channel", viper.GetString("spam_feed.channel")).
		Msg("starting penny")

	return myBot.Run()
}

// joinSpamFeedChannel finds the configured spam-feed channel by name and joins it.
func joinSpamFeedChannel(api slackclient.Client) error {
	channelName := viper.GetString("spam_feed.channel")
	if channelName == "" {
		log.Debug().Msg("no spam-feed channel configured, skipping auto-join")
		return nil
	}

	log.Debug().Str("channel", channelName).Msg("searching for spam-feed channel")
	cursor := ""
	for {
		params := &slack.GetConversationsParameters{
			ExcludeArchived: true,
			Limit:           200,
			Types:           []string{"public_channel", "private_channel"},
			Cursor:          cursor,
		}
		channels, nextCursor, err := api.GetConversations(params)
		if err != nil {
			return fmt.Errorf("listing conversations: %w", err)
		}
		for _, ch := range channels {
			if ch.NameNormalized == channelName {
				_, _, _, err := api.JoinConversation(ch.ID)
				if err != nil {
					return fmt.Errorf("joining channel %s: %w", channelName, err)
				}
				log.Info().Str("channel", channelName).Str("channel_id", ch.ID).Msg("joined spam-feed channel")
				return nil
			}
		}
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return fmt.Errorf("channel %q not found", channelName)
}

func setupServerFlags(c *cobra.Command) {
	c.PersistentFlags().IntP("port", "p", 3000, "The port on which the bot should bind.")
	viper.BindPFlag("server.port", c.PersistentFlags().Lookup("port"))
	viper.RegisterAlias("listen.port", "server.port")
	viper.SetDefault("server.port", 3000)

	c.PersistentFlags().String("db_hostname", "localhost", "The host for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.hostname", c.PersistentFlags().Lookup("db_hostname"))
	viper.RegisterAlias("db.host", "db.hostname")
	viper.SetDefault("db.hostname", "localhost")

	c.PersistentFlags().String("db_name", conf.Executable, "The name for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.name", c.PersistentFlags().Lookup("db_name"))
	viper.SetDefault("db.name", conf.Executable)

	c.PersistentFlags().String("db_username", "", "The username for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.username", c.PersistentFlags().Lookup("db_username"))
	viper.RegisterAlias("db.user", "db.username")
	viper.SetDefault("db.username", conf.Executable)

	c.PersistentFlags().String("db_password", "", "The password for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.password", c.PersistentFlags().Lookup("db_password"))
	viper.RegisterAlias("db.pass", "db.password")
}
