package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/spf13/viper"
	"github.com/xortim/penny/pkg/slackclient"
)

func setupViperConfig(t *testing.T, cfg map[string]interface{}) {
	for k, v := range cfg {
		viper.Set(k, v)
	}
	t.Cleanup(viper.Reset)
}

func TestJoinSpamFeedChannel(t *testing.T) {
	makeChannel := func(id, nameNormalized string) slack.Channel {
		return slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:              id,
					NameNormalized:  nameNormalized,
				},
			},
		}
	}

	tests := []struct {
		name           string
		config         map[string]interface{}
		mock           *slackclient.MockClient
		wantErr        bool
		wantErrContain string
	}{
		{
			name:   "no channel configured",
			config: map[string]interface{}{},
			mock:   &slackclient.MockClient{},
		},
		{
			name:   "empty channel configured",
			config: map[string]interface{}{"spam_feed.channel": ""},
			mock:   &slackclient.MockClient{},
		},
		{
			name:   "channel found on first page",
			config: map[string]interface{}{"spam_feed.channel": "spam-feed"},
			mock: &slackclient.MockClient{
				GetConversationsFn: func(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
					return []slack.Channel{
						makeChannel("C111", "general"),
						makeChannel("C222", "spam-feed"),
					}, "", nil
				},
				JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
					if channelID != "C222" {
						t.Errorf("expected JoinConversation with C222, got %s", channelID)
					}
					return nil, "", nil, nil
				},
			},
		},
		{
			name:   "channel found on second page",
			config: map[string]interface{}{"spam_feed.channel": "spam-feed"},
			mock: func() *slackclient.MockClient {
				callCount := 0
				return &slackclient.MockClient{
					GetConversationsFn: func(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
						callCount++
						if callCount == 1 {
							return []slack.Channel{
								makeChannel("C111", "general"),
								makeChannel("C333", "random"),
							}, "next-cursor", nil
						}
						return []slack.Channel{
							makeChannel("C222", "spam-feed"),
						}, "", nil
					},
					JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
						if channelID != "C222" {
							t.Errorf("expected JoinConversation with C222, got %s", channelID)
						}
						return nil, "", nil, nil
					},
				}
			}(),
		},
		{
			name:   "channel not found",
			config: map[string]interface{}{"spam_feed.channel": "nonexistent"},
			mock: &slackclient.MockClient{
				GetConversationsFn: func(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
					return []slack.Channel{
						makeChannel("C111", "general"),
					}, "", nil
				},
			},
			wantErr:        true,
			wantErrContain: `"nonexistent" not found`,
		},
		{
			name:   "GetConversations error",
			config: map[string]interface{}{"spam_feed.channel": "spam-feed"},
			mock: &slackclient.MockClient{
				GetConversationsFn: func(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
					return nil, "", errors.New("api timeout")
				},
			},
			wantErr:        true,
			wantErrContain: "listing conversations",
		},
		{
			name:   "JoinConversation error",
			config: map[string]interface{}{"spam_feed.channel": "spam-feed"},
			mock: &slackclient.MockClient{
				GetConversationsFn: func(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
					return []slack.Channel{
						makeChannel("C222", "spam-feed"),
					}, "", nil
				},
				JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
					return nil, "", nil, errors.New("permission denied")
				},
			},
			wantErr:        true,
			wantErrContain: "joining channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			err := joinSpamFeedChannel(tt.mock)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
