package help

import (
	"fmt"

	"github.com/gadget-bot/gadget/router"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

// GetSlashCommandRoutes returns the slash command routes for the help gadget.
func GetSlashCommandRoutes() []router.SlashCommandRoute {
	return []router.SlashCommandRoute{
		{
			Route: router.Route{
				Name:        "help.help",
				Description: "Show what Penny does and how to get help",
				Help:        "/help",
			},
			Command:           "/help",
			ImmediateResponse: formatHelp(),
			Plugin: func(r router.Router, route router.Route, api slack.Client, cmd slack.SlashCommand) {
				// No-op: all content is delivered via ImmediateResponse.
			},
		},
	}
}

// formatHelp builds the help text from viper config.
func formatHelp() string {
	emoji := viper.GetString("spam_feed.emoji")

	text := fmt.Sprintf(
		"*Penny* is a community moderation bot that monitors for the :%s: reaction to detect and remove spam messages.",
		emoji,
	)

	if channelID := viper.GetString("spam_feed.assistance_channel_id"); channelID != "" {
		text += fmt.Sprintf("\n\nNeed help or have questions? Visit <#%s>.", channelID)
	}

	return text
}
