package hallmonitor

import (
	"fmt"
	"strconv"

	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/spf13/viper"
	"github.com/xortim/penny/pkg/conversations"
	"github.com/xortim/penny/pkg/parsers"
)

const (
	BOT_MESSAGE_TYPE       = "bot_message"
	REACJI_USERNAME        = "Reacji Channeler"
	ACTIVITY_LOW_WATERMARK = 10

	OP_REMOVAL_REPLY = `Your message was reported by the community as SPAM and I've removed this post. Please join #admin-assistance channel if you have questions.`
)

func monitorSpamFeedMessages() *router.ChannelMessageRoute {
	var pluginRoute router.ChannelMessageRoute
	pluginRoute.Name = "hallmonitor.monitorSpamFeed"
	pluginRoute.Pattern = `.*`
	pluginRoute.Plugin = handleSpamFeedMessage
	return &pluginRoute
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
		print("there was error when getting the conversation information: ")
		println(err.Error())
	}
	if channelInfo.NameNormalized != viper.GetString("spam_feed.channel") {
		return
	}

	spamFeedMsgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
	spamFeedMsg, err := conversations.MsgRefToMessage(spamFeedMsgRef, api)
	if err != nil {
		print("could not get a message object from the message reference: ")
		println(err.Error())
		return
	}

	if len(viper.GetString("spam_feed.reaction_emoji")) != 0 {
		err = api.AddReaction(viper.GetString("spam_feed.reaction_emoji"), spamFeedMsgRef)
		if err != nil {
			println(err.Error())
		}
	}

	if len(viper.GetString("spam_feed.reacji_response")) != 0 {
		_, _, err = conversations.ThreadedReplyToMsg(spamFeedMsg, viper.GetString("spam_feed.reacji_response"), api)
		if err != nil {

			println(err.Error())
		}
	}

	opMsgRef := parsers.NewRefToMessageFromPermalink(message)
	opMsg, err := conversations.MsgRefToMessage(opMsgRef, api)
	if err != nil {
		print("could not get a message object from the message reference: ")
		println(err.Error())
	}

	removed := false
	score, reasons, err := AnomalyScore(opMsgRef, api)
	if err != nil {
		print("there was an error when calculating the anomaly score: ")
		println(err.Error())
	}

	if score >= viper.GetInt("spam_feed.max_anomaly_score") {
		_, _, err = conversations.ThreadedReplyToMsg(opMsg, OP_REMOVAL_REPLY, api)
		if err != nil {
			print("there was an error when replying to OP message: ")
			println(err.Error())
		}
		userTokenApi := slack.New(viper.GetString("slack.user_oauth_token"))
		_, _, err = userTokenApi.DeleteMessage(opMsg.Channel, opMsg.Timestamp)
		if err != nil {
			println(err.Error())
		}
		removed = true
	} else {
		if len(viper.GetString("spam_feed.op_warning")) != 0 {
			_, _, err = conversations.ThreadedReplyToMsg(opMsg, viper.GetString("spam_feed.op_warning"), api)
			if err != nil {
				print("there was an error when replying to OP message: ")
				println(err.Error())
			}
		}
	}

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
		_, _, err = conversations.ThreadedReplyToMsg(spamFeedMsg, debugResponse, api)
		if err != nil {
			print("there was an error when replying to the spam-feed message: ")
			println(err.Error())
		}
	}
}

// AnomalyScore returns the calculated anomaly score for the provided ItemRef
func AnomalyScore(ref slack.ItemRef, api slack.Client) (int, []string, error) {
	reasons := make([]string, 0)
	score := viper.GetInt("spam_feed.anomaly_scores.reported")
	if score != 0 {
		reasons = append(reasons, fmt.Sprintf("reported by the community as being spammy: %d", viper.GetInt("spam_feed.anomaly_scores.reported")))
	}

	opMsg, err := conversations.MsgRefToMessage(ref, api)
	if err != nil {
		return score, reasons, err
	}

	activityScore, err := userActivityScore(opMsg.User)
	if err != nil {
		print("an error occured when retriving the user activity: ")
		println(err.Error())
	}
	if activityScore != 0 {
		score += activityScore
		reasons = append(reasons, fmt.Sprintf("below the public activity low watermark: %d", activityScore))
		log.Debug().Str("anomaly_score", strconv.Itoa(score))
	}

	tzScore, err := userTzScore(opMsg.User, api)
	if err != nil {
		println(err.Error())
	}
	if tzScore != 0 {
		score += tzScore
		reasons = append(reasons, fmt.Sprintf("outside of the community timezone: %d", tzScore))
		log.Debug().Str("anomaly_score", strconv.Itoa(score))
	}

	return score, reasons, nil
}

// userActivityScore performs a public activity search for the specified user and returns the configured anomaly score if the total results are below the low watermark
// this must use the user oauth token so it initializes a new slack client each time
func userActivityScore(uid string) (int, error) {
	api := slack.New(viper.GetString("slack.user_oauth_token"))
	if viper.GetInt("spam_feed.activity_low_watermark") == 0 {
		return 0, nil
	}

	searchQuery := fmt.Sprintf("from:@%s", uid)

	results, err := api.SearchMessages(searchQuery, slack.NewSearchParameters())
	if err != nil {
		return 0, err
	}

	if results.TotalCount < viper.GetInt("spam_feed.activity_low_watermark") {
		return viper.GetInt("spam_feed.anomaly_scores.low_activity"), nil
	}

	return 0, nil
}

func userTzScore(uid string, api slack.Client) (int, error) {
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
