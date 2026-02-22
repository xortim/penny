package slackclient

import "github.com/slack-go/slack"

// Client is an interface over the Slack API methods used by penny.
// *slack.Client satisfies this interface at compile time.
type Client interface {
	GetConversationInfo(channel string, includeLocale bool) (*slack.Channel, error)
	JoinConversation(channelID string) (*slack.Channel, string, []string, error)
	GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	AddReaction(name string, item slack.ItemRef) error
	GetUserInfo(user string) (*slack.User, error)
	SearchMessages(query string, params slack.SearchParameters) (*slack.SearchMessages, error)
	DeleteMessage(channel, messageTimestamp string) (string, string, error)
	GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
}
