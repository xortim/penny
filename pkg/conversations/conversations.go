package conversations

import "github.com/slack-go/slack"

// ThreadedReply joins `ref.Channel` and creates or adds to the message's thread.
func ThreadedReply(ref slack.ItemRef, message string, api slack.Client) (string, string, error) {
	_, _, _, err := api.JoinConversation(ref.Channel)
	if err != nil {
		return "", "", err
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
