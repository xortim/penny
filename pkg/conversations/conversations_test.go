package conversations

import (
	"errors"
	"reflect"
	"testing"

	"github.com/slack-go/slack"
	"github.com/xortim/penny/pkg/slackclient"
)

// noopJoin is a JoinConversation stub that always succeeds.
func noopJoin(channelID string) (*slack.Channel, string, []string, error) {
	return &slack.Channel{}, "", nil, nil
}

// historyWith returns a GetConversationHistory stub that returns a single message with the given timestamp.
func historyWith(ts string) func(*slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	return func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
		return &slack.GetConversationHistoryResponse{
			Messages: []slack.Message{
				{Msg: slack.Msg{Timestamp: ts}},
			},
		}, nil
	}
}

func TestWhoReactedWith(t *testing.T) {
	type args struct {
		msg          slack.Message
		reactionName string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Message without target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{},
		},
		{
			name: "Message with target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"U12345"},
		},
		{
			name: "Message with target reaction and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"U12345"},
		},
		{
			name: "Message with multiple target reactions and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 2,
								Users: []string{"U12345", "U67890"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"U12345", "U67890"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WhoReactedWith(tt.args.msg, tt.args.reactionName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhoReactedWith() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhoReactedWithAsMention(t *testing.T) {
	type args struct {
		msg          slack.Message
		reactionName string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Message without target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{},
		},
		{
			name: "Message with target reaction",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"<@U12345>"},
		},
		{
			name: "Message with target reaction and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 1,
								Users: []string{"U12345"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"<@U12345>"},
		},
		{
			name: "Message with multiple target reactions and others",
			args: args{
				msg: slack.Message{
					Msg: slack.Msg{
						Reactions: []slack.ItemReaction{
							{
								Name:  "thumbsup",
								Count: 1,
								Users: []string{"U54321"},
							},
							{
								Name:  "nospam",
								Count: 2,
								Users: []string{"U12345", "U67890"},
							},
						},
					},
				},
				reactionName: "nospam",
			},
			want: []string{"<@U12345>", "<@U67890>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WhoReactedWithAsMention(tt.args.msg, tt.args.reactionName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhoReactedWithAsMention() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMsgRefToMessage(t *testing.T) {
	ref := slack.NewRefToMessage("C123", "1234567890.000100")

	tests := []struct {
		name      string
		mock      *slackclient.MockClient
		wantErr   string
		wantChan  string
		wantTS    string
	}{
		{
			name: "JoinConversation error",
			mock: &slackclient.MockClient{
				JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
					return nil, "", nil, errors.New("join failed")
				},
			},
			wantErr: "join failed",
		},
		{
			name: "GetConversationHistory error",
			mock: &slackclient.MockClient{
				JoinConversationFn: noopJoin,
				GetConversationHistoryFn: func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
					return nil, errors.New("history failed")
				},
			},
			wantErr: "history failed",
		},
		{
			name: "No messages in response",
			mock: &slackclient.MockClient{
				JoinConversationFn: noopJoin,
				GetConversationHistoryFn: func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
					return &slack.GetConversationHistoryResponse{Messages: []slack.Message{}}, nil
				},
			},
			wantErr: "message not found",
		},
		{
			name: "Timestamp mismatch",
			mock: &slackclient.MockClient{
				JoinConversationFn: noopJoin,
				GetConversationHistoryFn: func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
					return &slack.GetConversationHistoryResponse{
						Messages: []slack.Message{
							{Msg: slack.Msg{Timestamp: "9999999999.000000"}},
						},
					}, nil
				},
			},
			wantErr: "message not found",
		},
		{
			name: "Happy path",
			mock: &slackclient.MockClient{
				JoinConversationFn:       noopJoin,
				GetConversationHistoryFn: historyWith("1234567890.000100"),
			},
			wantChan: "C123",
			wantTS:   "1234567890.000100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MsgRefToMessage(ref, tt.mock)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("MsgRefToMessage() error = %v, wantErr %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("MsgRefToMessage() unexpected error: %v", err)
			}
			if got.Channel != tt.wantChan {
				t.Errorf("MsgRefToMessage() Channel = %q, want %q", got.Channel, tt.wantChan)
			}
			if got.Timestamp != tt.wantTS {
				t.Errorf("MsgRefToMessage() Timestamp = %q, want %q", got.Timestamp, tt.wantTS)
			}
		})
	}
}

func TestThreadedReplyToMsg(t *testing.T) {
	tests := []struct {
		name       string
		msg        slack.Message
		postMsgErr error
		wantChan   string
		wantErr    bool
	}{
		{
			name: "Non-threaded message uses Timestamp",
			msg: slack.Message{
				Msg: slack.Msg{Timestamp: "1111111111.000100", Channel: "C_TEST"},
			},
			wantChan: "C_TEST",
		},
		{
			name: "Threaded message uses ThreadTimestamp",
			msg: slack.Message{
				Msg: slack.Msg{
					Timestamp:       "1111111111.000100",
					ThreadTimestamp: "1111111110.000100",
					Channel:         "C_TEST",
				},
			},
			wantChan: "C_TEST",
		},
		{
			name: "PostMessage error propagated",
			msg: slack.Message{
				Msg: slack.Msg{Timestamp: "1111111111.000100", Channel: "C_TEST"},
			},
			postMsgErr: errors.New("post failed"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			calledChan := ""
			mock := &slackclient.MockClient{
				PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
					called = true
					calledChan = channelID
					return channelID, "1111111111.000100", tt.postMsgErr
				},
			}

			_, _, err := ThreadedReplyToMsg(tt.msg, "reply text", mock)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ThreadedReplyToMsg() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ThreadedReplyToMsg() unexpected error: %v", err)
			}
			if !called {
				t.Errorf("ThreadedReplyToMsg() PostMessage not called")
			}
			if calledChan != tt.wantChan {
				t.Errorf("ThreadedReplyToMsg() PostMessage channel = %q, want %q", calledChan, tt.wantChan)
			}
		})
	}
}

func TestThreadReplyToMessage(t *testing.T) {
	const (
		channelID = "C123"
		threadTS  = "1234567890.000100"
		replyTS   = "1234567891.000200"
	)

	parentMsg := slack.Message{Msg: slack.Msg{Timestamp: threadTS}}
	replyMsg := slack.Message{Msg: slack.Msg{Timestamp: replyTS}}

	tests := []struct {
		name     string
		mock     *slackclient.MockClient
		wantErr  string
		wantTS   string
		wantChan string
	}{
		{
			name: "JoinConversation error",
			mock: &slackclient.MockClient{
				JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
					return nil, "", nil, errors.New("join failed")
				},
			},
			wantErr: "join failed",
		},
		{
			name: "GetConversationReplies error",
			mock: &slackclient.MockClient{
				JoinConversationFn: noopJoin,
				GetConversationRepliesFn: func(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
					return nil, false, "", errors.New("replies failed")
				},
			},
			wantErr: "replies failed",
		},
		{
			name: "Reply not found in results",
			mock: &slackclient.MockClient{
				JoinConversationFn: noopJoin,
				GetConversationRepliesFn: func(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
					// API always includes parent; reply is missing
					return []slack.Message{parentMsg}, false, "", nil
				},
			},
			wantErr: "reply not found",
		},
		{
			name: "Happy path",
			mock: &slackclient.MockClient{
				JoinConversationFn: noopJoin,
				GetConversationRepliesFn: func(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
					return []slack.Message{parentMsg, replyMsg}, false, "", nil
				},
			},
			wantTS:   replyTS,
			wantChan: channelID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ThreadReplyToMessage(channelID, threadTS, replyTS, tt.mock)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("ThreadReplyToMessage() error = %v, wantErr %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ThreadReplyToMessage() unexpected error: %v", err)
			}
			if got.Timestamp != tt.wantTS {
				t.Errorf("ThreadReplyToMessage() Timestamp = %q, want %q", got.Timestamp, tt.wantTS)
			}
			if got.Channel != tt.wantChan {
				t.Errorf("ThreadReplyToMessage() Channel = %q, want %q", got.Channel, tt.wantChan)
			}
		})
	}
}

func TestThreadedReplyToMsgRef(t *testing.T) {
	ref := slack.NewRefToMessage("C123", "1234567890.000100")

	t.Run("MsgRefToMessage error propagated", func(t *testing.T) {
		mock := &slackclient.MockClient{
			JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
				return nil, "", nil, errors.New("join failed")
			},
		}
		_, _, err := ThreadedReplyToMsgRef(ref, "reply", mock)
		if err == nil || err.Error() != "join failed" {
			t.Errorf("ThreadedReplyToMsgRef() error = %v, want 'join failed'", err)
		}
	})

	t.Run("Success path returns channel and ts from PostMessage", func(t *testing.T) {
		mock := &slackclient.MockClient{
			JoinConversationFn:       noopJoin,
			GetConversationHistoryFn: historyWith("1234567890.000100"),
			PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
				return "C123", "1234567890.000100", nil
			},
		}
		ch, ts, err := ThreadedReplyToMsgRef(ref, "reply", mock)
		if err != nil {
			t.Fatalf("ThreadedReplyToMsgRef() unexpected error: %v", err)
		}
		if ch != "C123" || ts != "1234567890.000100" {
			t.Errorf("ThreadedReplyToMsgRef() = (%q, %q), want (C123, 1234567890.000100)", ch, ts)
		}
	})
}
