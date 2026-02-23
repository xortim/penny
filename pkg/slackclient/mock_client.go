package slackclient

import "github.com/slack-go/slack"

// MockClient is a configurable mock of Client for use in tests.
// Set only the Fn fields you need; unconfigured methods panic to signal unexpected calls.
type MockClient struct {
	GetConversationInfoFn    func(input *slack.GetConversationInfoInput) (*slack.Channel, error)
	JoinConversationFn       func(channelID string) (*slack.Channel, string, []string, error)
	GetConversationHistoryFn func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	PostMessageFn            func(channelID string, options ...slack.MsgOption) (string, string, error)
	AddReactionFn            func(name string, item slack.ItemRef) error
	GetUserInfoFn            func(user string) (*slack.User, error)
	SearchMessagesFn         func(query string, params slack.SearchParameters) (*slack.SearchMessages, error)
	DeleteMessageFn          func(channel, messageTimestamp string) (string, string, error)
	GetConversationsFn       func(params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
}

func (m *MockClient) GetConversationInfo(input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	return m.GetConversationInfoFn(input)
}

func (m *MockClient) JoinConversation(channelID string) (*slack.Channel, string, []string, error) {
	return m.JoinConversationFn(channelID)
}

func (m *MockClient) GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	return m.GetConversationHistoryFn(params)
}

func (m *MockClient) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return m.PostMessageFn(channelID, options...)
}

func (m *MockClient) AddReaction(name string, item slack.ItemRef) error {
	return m.AddReactionFn(name, item)
}

func (m *MockClient) GetUserInfo(user string) (*slack.User, error) {
	return m.GetUserInfoFn(user)
}

func (m *MockClient) SearchMessages(query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
	return m.SearchMessagesFn(query, params)
}

func (m *MockClient) DeleteMessage(channel, messageTimestamp string) (string, string, error) {
	return m.DeleteMessageFn(channel, messageTimestamp)
}

func (m *MockClient) GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	return m.GetConversationsFn(params)
}
