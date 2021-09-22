package hallmonitor

import (
	"github.com/gadget-bot/gadget/router"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/spf13/viper"
	"github.com/xortim/penny/pkg/parsers"
)

const (
	BOT_MESSAGE_TYPE = "bot_message"
	REACJI_USERNAME  = "Reacji Channeler"
)

func monitorSpamFeedMessages() *router.ChannelMessageRoute {
	var pluginRoute router.ChannelMessageRoute
	pluginRoute.Name = "hallmonitor.monitorSpamFeed"
	pluginRoute.Pattern = `.*`
	pluginRoute.Plugin = handleSpamFeedMessage
	return &pluginRoute
}

func GetChannelMessageRoutes() []router.ChannelMessageRoute {
	return []router.ChannelMessageRoute{
		*monitorSpamFeedMessages(),
	}
}

func handleSpamFeedMessage(router router.Router, route router.Route, api slack.Client, ev slackevents.MessageEvent, message string) {
	// only look at the original, unfurled message
	if ev.SubType != BOT_MESSAGE_TYPE && ev.Username != REACJI_USERNAME {
		return
	}

	// // this stopped working as some point.. ev.Icons.IconEmoji is empty
	// // only look at the messages the messages with the configured reaction from Reacji
	// if ev.Icons != nil && ev.Icons.IconEmoji != viper.GetString("spam_feed.emoji") {
	// 	return
	// }

	// only look at messages in the correct channel.
	// this performs another API request so we do this later than the other checks
	channelInfo, err := api.GetConversationInfo(ev.Channel, false)
	if err != nil {
		println(err.Error())
	}
	if channelInfo.NameNormalized != viper.GetString("spam_feed.channel") {
		return
	}

	spamFeedMsgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)

	if len(viper.GetString("spam_feed.reaction_emoji")) != 0 {
		err = api.AddReaction(viper.GetString("spam_feed.reaction_emoji"), spamFeedMsgRef)
		if err != nil {
			println(err.Error())
		}
	}

	if len(viper.GetString("spam_feed.response")) != 0 {
		_, _, err = threadedReply(spamFeedMsgRef, viper.GetString("spam_feed.response"), api)
		if err != nil {
			println(err.Error())
		}
	}

	opRef := parsers.NewRefToMessageFromPermalink(message)
	_, _, err = threadedReply(opRef, "This message has been flagged by our community as SPAM. The admins have been notified.", api)
	if err != nil {
		println(err.Error())
	}

	_, _, _ = threadedReply(opRef, "Second reply", api)
}

func threadedReply(ref slack.ItemRef, message string, api slack.Client) (string, string, error) {
	_, _, _, err := api.JoinConversation(ref.Channel)
	if err != nil {
		println(err.Error())
	}

	op, err := api.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: ref.Channel,
		Latest:    ref.Timestamp,
		Limit:     1,
		Inclusive: true,
	})
	if err != nil {
		return "", "", err
	}

	// use the correct timestamp for starting or posting to a
	// thread. otherwise the bot _could_ modify the original message
	// which causes it to show up in the top-level conversation
	ts := ref.Timestamp
	if len(op.Messages[0].ThreadTimestamp) != 0 {
		ts = op.Messages[0].ThreadTimestamp
	}

	return api.PostMessage(
		ref.Channel,
		slack.MsgOptionTS(ts),
		slack.MsgOptionText(message, false),
	)
}
