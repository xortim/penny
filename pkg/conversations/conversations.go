package conversations

import (
	"fmt"

	"github.com/slack-go/slack"
)

// ThreadedReplyToMsgRef delegates to MsgRefToMessage and ThreadedReplyToMsg
func ThreadedReplyToMsgRef(ref slack.ItemRef, reply string, api slack.Client) (string, string, error) {
	message, err := MsgRefToMessage(ref, api)
	if err != nil {
		return "", "", err
	}

	return ThreadedReplyToMsg(message, reply, api)
}

// MsgRefToMessage joins the channel in the message reference and returns the found Message struct
func MsgRefToMessage(ref slack.ItemRef, api slack.Client) (slack.Message, error) {
	message := &slack.Message{}
	_, _, _, err := api.JoinConversation(ref.Channel)
	if err != nil {
		return *message, err
	}

	response, err := api.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: ref.Channel,
		Latest:    ref.Timestamp,
		Limit:     1,
		Inclusive: true,
	})
	if err != nil {
		return *message, err
	}

	if len(response.Messages) != 1 {
		return *message, fmt.Errorf("message not found")
	}

	message = &response.Messages[0]
	message.Channel = ref.Channel

	return *message, nil
}

func ThreadedReplyToMsg(msg slack.Message, reply string, api slack.Client) (string, string, error) {
	// Use the correct timestamp for starting or posting to a
	// thread. Otherwise the bot _could_ modify the message
	// which causes it to show up in the top-level conversation.
	// This happens if you try to reply to a message already in
	// a thread.
	ts := msg.Timestamp
	if len(msg.ThreadTimestamp) != 0 {
		ts = msg.ThreadTimestamp
	}

	return api.PostMessage(
		msg.Channel,
		slack.MsgOptionTS(ts),
		slack.MsgOptionText(reply, false),
	)
}
