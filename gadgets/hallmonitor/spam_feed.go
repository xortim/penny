package hallmonitor

import (
	"fmt"
	"strings"

	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/spf13/viper"
	"github.com/xortim/penny/pkg/conversations"
	"github.com/xortim/penny/pkg/parsers"
	"github.com/xortim/penny/pkg/slackclient"
)

const (
	BOT_MESSAGE_TYPE = "bot_message"
	REACJI_USERNAME  = "Reacji Channeler"
)

func removalReply() string {
	message := "Your message was reported by the community as SPAM and I've removed this post."

	if len(viper.GetString("spam_feed.assistance_channel_id")) != 0 {
		message = fmt.Sprintf("%s. Please join <#%s> if you have questions.", message, viper.GetString("spam_feed.assistance_channel_id"))
	}
	return message
}

func monitorSpamFeedMessages() *router.ChannelMessageRoute {
	var pluginRoute router.ChannelMessageRoute
	pluginRoute.Name = "hallmonitor.monitorSpamFeed"
	pluginRoute.Pattern = `.*`
	pluginRoute.Plugin = handleSpamFeedMessage
	return &pluginRoute
}

// handleSpamFeedMessage is the Gadget-registered handler. Its signature is fixed by the framework.
func handleSpamFeedMessage(r router.Router, route router.Route, api slack.Client, ev slackevents.MessageEvent, message string) {
	userApi := slack.New(viper.GetString("slack.user_oauth_token"))
	ProcessSpamFeedMessage(r, route, &api, userApi, ev, message)
}

// ProcessSpamFeedMessage contains the testable core logic extracted from handleSpamFeedMessage.
// Exported so that integration tests can inject both API clients.
func ProcessSpamFeedMessage(r router.Router, route router.Route, api slackclient.Client, userApi slackclient.Client, ev slackevents.MessageEvent, message string) {
	logger := log.With().Str("channel_id", ev.Channel).Str("event_ts", ev.TimeStamp).Logger()

	// only look at the original, unfurled message
	if ev.SubType != BOT_MESSAGE_TYPE && ev.Username != REACJI_USERNAME {
		return
	}

	// only look at messages in the correct channel.
	channelInfo, err := api.GetConversationInfo(&slack.GetConversationInfoInput{ChannelID: ev.Channel})
	if err != nil {
		logger.Error().Err(err).Msg("failed to get conversation info")
	}
	if channelInfo.NameNormalized != viper.GetString("spam_feed.channel") {
		return
	}

	logger.Info().Msg("processing spam feed message")

	spamFeedMsgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
	spamFeedMsg, err := conversations.MsgRefToMessage(spamFeedMsgRef, api)
	if err != nil {
		logger.Error().Err(err).Msg("failed to resolve spam-feed message reference")
		return
	}

	opMsgRef, opThreaded := parsers.NewRefToMessageFromPermalink(strings.Trim(message, "<>"))
	if opThreaded {
		_, _, _ = conversations.ThreadedReplyToMsg(spamFeedMsg, "I currently don't handle threaded replies.", api)
		// TODO: https://api.slack.com/messaging/retrieving#pulling_threads
		return
	}

	opMsg, err := conversations.MsgRefToMessage(opMsgRef, api)
	if err != nil {
		_, _, _ = conversations.ThreadedReplyToMsg(spamFeedMsg, "I couldn't retrieve the original message from the Slack API.", api)
	}

	reporters := conversations.WhoReactedWithAsMention(opMsg, viper.GetString("spam_feed.emoji"))

	// acknowledge the users that reported message
	ack := ""
	if len(reporters) != 0 {
		ack = fmt.Sprintf("Thanks %s! ", strings.Join(reporters, ","))
	}
	if len(viper.GetString("spam_feed.reacji_response")) != 0 {
		ack = fmt.Sprintf("%s%s", ack, viper.GetString("spam_feed.reacji_response"))
	}
	_, _, err = conversations.ThreadedReplyToMsg(spamFeedMsg, ack, api)
	if err != nil {
		logger.Error().Err(err).Msg("failed to send acknowledgment reply")
	}

	if opMsg.User == r.BotUID {
		_, _, err = conversations.ThreadedReplyToMsg(spamFeedMsg, "Hey! That's not nice.", api)
		if err != nil {
			logger.Error().Err(err).Msg("failed to reply to spam feed message")
		}
		return
	}

	removed := false
	score, reasons, err := anomalyScoreInternal(opMsgRef, api, userApi, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to calculate anomaly score")
	}

	if score >= viper.GetInt("spam_feed.max_anomaly_score") {
		logger.Info().Int("score", score).Int("threshold", viper.GetInt("spam_feed.max_anomaly_score")).Msg("message removed")
		_, _, err = conversations.ThreadedReplyToMsg(opMsg, removalReply(), api)
		if err != nil {
			logger.Error().Err(err).Msg("failed to warn OP before removal")
		}
		_, _, err = userApi.DeleteMessage(opMsg.Channel, opMsg.Timestamp)
		if err != nil {
			logger.Error().Err(err).Str("op_channel", opMsg.Channel).Str("op_ts", opMsg.Timestamp).Msg("failed to delete message")
		}
		removed = true
	} else {
		logger.Info().Int("score", score).Int("threshold", viper.GetInt("spam_feed.max_anomaly_score")).Msg("below threshold")
		if len(viper.GetString("spam_feed.op_warning")) != 0 {
			_, _, err = conversations.ThreadedReplyToMsg(opMsg, viper.GetString("spam_feed.op_warning"), api)
			if err != nil {
				logger.Error().Err(err).Msg("failed to warn OP")
			}
		}
	}

	err = addAnomalyReaction(removed, spamFeedMsgRef, api)
	if err != nil {
		logger.Error().Err(err).Msg("failed to add anomaly reaction")
	}

	err = addDebugResponse(removed, score, reasons, spamFeedMsg, api)
	if err != nil {
		logger.Error().Err(err).Msg("failed to post debug response")
	}
}

// AnomalyScore returns the calculated anomaly score for the provided ItemRef.
// It is a thin wrapper that creates the user-token client and delegates to anomalyScoreInternal.
func AnomalyScore(ref slack.ItemRef, api slack.Client) (int, []string, error) {
	userApi := slack.New(viper.GetString("slack.user_oauth_token"))
	return anomalyScoreInternal(ref, &api, userApi, log.Logger)
}

func anomalyScoreInternal(ref slack.ItemRef, api slackclient.Client, userApi slackclient.Client, logger zerolog.Logger) (int, []string, error) {
	reasons := make([]string, 0)
	score := viper.GetInt("spam_feed.anomaly_scores.reported")
	if score != 0 {
		reasons = append(reasons, fmt.Sprintf("reported by the community as being spammy: %d", viper.GetInt("spam_feed.anomaly_scores.reported")))
	}

	opMsg, err := conversations.MsgRefToMessage(ref, api)
	if err != nil {
		return score, reasons, err
	}

	activityScore, err := userActivityScore(opMsg.User, userApi)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve user activity")
	}
	if activityScore != 0 {
		score += activityScore
		reasons = append(reasons, fmt.Sprintf("below the public activity low watermark: %d", activityScore))
		logger.Debug().Int("anomaly_score", score).Msg("added activity score")
	}

	tzScore, err := userTzScore(opMsg.User, api)
	if err != nil {
		logger.Error().Err(err).Msg("failed to check user timezone")
	}
	if tzScore != 0 {
		score += tzScore
		reasons = append(reasons, fmt.Sprintf("outside of the community timezone: %d", tzScore))
		logger.Debug().Int("anomaly_score", score).Msg("added timezone score")
	}

	logger.Info().Int("anomaly_score", score).Int("reasons", len(reasons)).Msg("anomaly score calculated")
	return score, reasons, nil
}

// userActivityScore performs a public activity search for the specified user and returns
// the configured anomaly score if the total results are below the low watermark.
func userActivityScore(uid string, api slackclient.Client) (int, error) {
	if viper.GetInt("spam_feed.activity_low_watermark") == 0 {
		return 0, nil
	}

	searchQuery := fmt.Sprintf("after:2021/12/01 from:<@%s>", uid)

	results, err := api.SearchMessages(searchQuery, slack.NewSearchParameters())
	if err != nil {
		return 0, err
	}

	if results.TotalCount < viper.GetInt("spam_feed.activity_low_watermark") {
		return viper.GetInt("spam_feed.anomaly_scores.low_activity"), nil
	}

	return 0, nil
}

func userTzScore(uid string, api slackclient.Client) (int, error) {
	if len(viper.GetString("spam_feed.local_timezone")) == 0 {
		return 0, nil
	}

	user, err := api.GetUserInfo(uid)
	if err != nil {
		return 0, err
	}

	if user.TZ != viper.GetString("spam_feed.local_timezone") {
		return viper.GetInt("spam_feed.anomaly_scores.outside_tz"), nil
	}
	return 0, nil
}

func addAnomalyReaction(removed bool, msgRef slack.ItemRef, api slackclient.Client) error {
	emoji_conf := "spam_feed.reaction_emoji_miss"
	if removed {
		emoji_conf = "spam_feed.reaction_emoji_hit"
	}
	if len(viper.GetString(emoji_conf)) != 0 {
		err := api.AddReaction(viper.GetString(emoji_conf), msgRef)
		if err != nil {
			return err
		}
	}
	return nil
}

func addDebugResponse(removed bool, score int, reasons []string, msg slack.Message, api slackclient.Client) error {
	var err error
	if len(reasons) != 0 {
		debugResponse := "This is what I found about the OP:\n"
		for _, v := range reasons {
			debugResponse += fmt.Sprintf("- %s\n", v)
		}
		if removed {
			debugResponse += fmt.Sprintf("I removed the OP since the final anomaly score (%d/%d) was suspect enough.", score, viper.GetInt("spam_feed.max_anomaly_score"))
		} else {
			debugResponse += fmt.Sprintf("The final anomaly score (%d/%d) didn't result in a removal.", score, viper.GetInt("spam_feed.max_anomaly_score"))
		}
		_, _, err = conversations.ThreadedReplyToMsg(msg, debugResponse, api)
	}
	return err
}
