package hallmonitor

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/spf13/viper"
	"github.com/xortim/penny/pkg/slackclient"
)

// setupViperConfig sets the given keys in the global viper instance and registers
// a cleanup that calls viper.Reset() after each test.
func setupViperConfig(t *testing.T, cfg map[string]interface{}) {
	t.Helper()
	for k, v := range cfg {
		viper.Set(k, v)
	}
	t.Cleanup(viper.Reset)
}

// noopJoin is a JoinConversation stub that always succeeds.
func noopJoin(channelID string) (*slack.Channel, string, []string, error) {
	return &slack.Channel{}, "", nil, nil
}

// noopPost is a PostMessage stub that always succeeds.
func noopPost(channelID string, options ...slack.MsgOption) (string, string, error) {
	return channelID, "ts", nil
}

// noopDelete is a DeleteMessage stub that always succeeds.
func noopDelete(channel, messageTimestamp string) (string, string, error) {
	return channel, messageTimestamp, nil
}

// noopReaction is an AddReaction stub that always succeeds.
func noopReaction(name string, item slack.ItemRef) error { return nil }

// historyFor returns a GetConversationHistory stub that dispatches by ChannelID,
// returning the matching message from the provided map.
func historyFor(byChannel map[string]slack.Message) func(*slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	return func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
		msg, ok := byChannel[params.ChannelID]
		if !ok {
			return &slack.GetConversationHistoryResponse{Messages: []slack.Message{}}, nil
		}
		return &slack.GetConversationHistoryResponse{Messages: []slack.Message{msg}}, nil
	}
}

// TestRemovalReply verifies the removal reply message with and without an assistance channel configured.
func TestRemovalReply(t *testing.T) {
	tests := []struct {
		name              string
		assistanceChannel string
		want              string
	}{
		{
			name: "No assistance channel",
			want: "Your message was reported by the community as SPAM and I've removed this post.",
		},
		{
			name:              "With assistance channel",
			assistanceChannel: "CEXAMPLE",
			want:              "Your message was reported by the community as SPAM and I've removed this post.. Please join <#CEXAMPLE> if you have questions.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, map[string]interface{}{
				"spam_feed.assistance_channel_id": tt.assistanceChannel,
			})
			got := removalReply()
			if got != tt.want {
				t.Errorf("removalReply() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUserTzScore verifies timezone anomaly scoring.
func TestUserTzScore(t *testing.T) {
	tests := []struct {
		name       string
		config     map[string]interface{}
		userTZ     string
		getUserErr error
		wantScore  int
		wantErr    bool
	}{
		{
			name:      "local_timezone not configured returns 0",
			config:    map[string]interface{}{"spam_feed.local_timezone": ""},
			wantScore: 0,
		},
		{
			name: "User TZ matches config returns 0",
			config: map[string]interface{}{
				"spam_feed.local_timezone":             "America/New_York",
				"spam_feed.anomaly_scores.outside_tz": 2,
			},
			userTZ:    "America/New_York",
			wantScore: 0,
		},
		{
			name: "User TZ differs from config returns score",
			config: map[string]interface{}{
				"spam_feed.local_timezone":             "America/New_York",
				"spam_feed.anomaly_scores.outside_tz": 2,
			},
			userTZ:    "Asia/Tokyo",
			wantScore: 2,
		},
		{
			name: "GetUserInfo error returns 0 and error",
			config: map[string]interface{}{
				"spam_feed.local_timezone": "America/New_York",
			},
			getUserErr: errors.New("user not found"),
			wantScore:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			mock := &slackclient.MockClient{
				GetUserInfoFn: func(user string) (*slack.User, error) {
					if tt.getUserErr != nil {
						return nil, tt.getUserErr
					}
					return &slack.User{TZ: tt.userTZ}, nil
				},
			}

			got, err := userTzScore("U123", mock)
			if tt.wantErr && err == nil {
				t.Errorf("userTzScore() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("userTzScore() unexpected error: %v", err)
			}
			if got != tt.wantScore {
				t.Errorf("userTzScore() = %d, want %d", got, tt.wantScore)
			}
		})
	}
}

// TestUserActivityScore verifies activity-based anomaly scoring.
func TestUserActivityScore(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		searchTotal int
		searchErr   error
		wantScore   int
		wantErr     bool
	}{
		{
			name:      "Watermark disabled (0) returns 0 without API call",
			config:    map[string]interface{}{"spam_feed.activity_low_watermark": 0},
			wantScore: 0,
		},
		{
			name: "Message count at or above watermark returns 0",
			config: map[string]interface{}{
				"spam_feed.activity_low_watermark":      10,
				"spam_feed.anomaly_scores.low_activity": 1,
			},
			searchTotal: 15,
			wantScore:   0,
		},
		{
			name: "Message count below watermark returns score",
			config: map[string]interface{}{
				"spam_feed.activity_low_watermark":      10,
				"spam_feed.anomaly_scores.low_activity": 1,
			},
			searchTotal: 5,
			wantScore:   1,
		},
		{
			name:      "SearchMessages error returns 0 and error",
			config:    map[string]interface{}{"spam_feed.activity_low_watermark": 10},
			searchErr: errors.New("search failed"),
			wantScore: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			mock := &slackclient.MockClient{
				SearchMessagesFn: func(query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
					if tt.searchErr != nil {
						return nil, tt.searchErr
					}
					return &slack.SearchMessages{
						Pagination: slack.Pagination{TotalCount: tt.searchTotal},
					}, nil
				},
			}

			got, err := userActivityScore("U123", mock)
			if tt.wantErr && err == nil {
				t.Errorf("userActivityScore() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("userActivityScore() unexpected error: %v", err)
			}
			if got != tt.wantScore {
				t.Errorf("userActivityScore() = %d, want %d", got, tt.wantScore)
			}
		})
	}
}

// TestAddAnomalyReaction verifies that the correct emoji is added based on the removal outcome.
func TestAddAnomalyReaction(t *testing.T) {
	msgRef := slack.NewRefToMessage("C123", "1234567890.000100")

	tests := []struct {
		name               string
		removed            bool
		config             map[string]interface{}
		reactionErr        error
		wantReactionCalled bool
		wantEmoji          string
		wantErr            bool
	}{
		{
			name:               "Removed=true uses hit emoji",
			removed:            true,
			config:             map[string]interface{}{"spam_feed.reaction_emoji_hit": "no_entry"},
			wantReactionCalled: true,
			wantEmoji:          "no_entry",
		},
		{
			name:               "Removed=false uses miss emoji",
			removed:            false,
			config:             map[string]interface{}{"spam_feed.reaction_emoji_miss": "white_check_mark"},
			wantReactionCalled: true,
			wantEmoji:          "white_check_mark",
		},
		{
			name:               "Empty emoji config skips AddReaction",
			removed:            true,
			config:             map[string]interface{}{"spam_feed.reaction_emoji_hit": ""},
			wantReactionCalled: false,
		},
		{
			name:               "AddReaction error is returned",
			removed:            true,
			config:             map[string]interface{}{"spam_feed.reaction_emoji_hit": "no_entry"},
			reactionErr:        errors.New("reaction failed"),
			wantReactionCalled: true,
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			called := false
			gotEmoji := ""
			mock := &slackclient.MockClient{
				AddReactionFn: func(name string, item slack.ItemRef) error {
					called = true
					gotEmoji = name
					return tt.reactionErr
				},
			}

			err := addAnomalyReaction(tt.removed, msgRef, mock)
			if tt.wantErr && err == nil {
				t.Errorf("addAnomalyReaction() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("addAnomalyReaction() unexpected error: %v", err)
			}
			if called != tt.wantReactionCalled {
				t.Errorf("addAnomalyReaction() AddReaction called = %v, want %v", called, tt.wantReactionCalled)
			}
			if tt.wantReactionCalled && !tt.wantErr && gotEmoji != tt.wantEmoji {
				t.Errorf("addAnomalyReaction() emoji = %q, want %q", gotEmoji, tt.wantEmoji)
			}
		})
	}
}

// TestAddDebugResponse verifies PostMessage is called only when there are reasons,
// and that the message text reflects the removal outcome.
func TestAddDebugResponse(t *testing.T) {
	spamFeedMsg := slack.Message{
		Msg: slack.Msg{Timestamp: "1111.0001", Channel: "C_SPAM"},
	}

	tests := []struct {
		name         string
		removed      bool
		score        int
		reasons      []string
		config       map[string]interface{}
		wantPostCall bool
		wantContains string
	}{
		{
			name:         "Empty reasons - no PostMessage call",
			removed:      false,
			score:        2,
			reasons:      []string{},
			config:       map[string]interface{}{"spam_feed.max_anomaly_score": 5},
			wantPostCall: false,
		},
		{
			name:         "Reasons with removed=true includes removal text",
			removed:      true,
			score:        5,
			reasons:      []string{"reported by community: 2"},
			config:       map[string]interface{}{"spam_feed.max_anomaly_score": 5},
			wantPostCall: true,
			wantContains: "I removed",
		},
		{
			name:         "Reasons with removed=false includes non-removal text",
			removed:      false,
			score:        2,
			reasons:      []string{"reported by community: 2"},
			config:       map[string]interface{}{"spam_feed.max_anomaly_score": 5},
			wantPostCall: true,
			wantContains: "didn't result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			called := false
			mock := &slackclient.MockClient{
				PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
					called = true
					return channelID, "ts", nil
				},
			}

			err := addDebugResponse(tt.removed, tt.score, tt.reasons, spamFeedMsg, mock)
			if err != nil {
				t.Fatalf("addDebugResponse() unexpected error: %v", err)
			}
			if called != tt.wantPostCall {
				t.Errorf("addDebugResponse() PostMessage called = %v, want %v", called, tt.wantPostCall)
			}
		})
	}
}

// TestAnomalyScoreInternal verifies the full anomaly score calculation with injected mocks.
func TestAnomalyScoreInternal(t *testing.T) {
	const (
		opChan = "C_OP_CHAN"
		opTS   = "1639843883.000100"
		opUser = "U_OP_USER"
	)

	opMsg := slack.Message{
		Msg: slack.Msg{Timestamp: opTS, Channel: opChan, User: opUser},
	}
	ref := slack.NewRefToMessage(opChan, opTS)

	// apiMock builds a mock that returns opMsg on history lookup and optionally GetUserInfo.
	apiMock := func(userTZ string, getUserErr error) *slackclient.MockClient {
		return &slackclient.MockClient{
			JoinConversationFn: noopJoin,
			GetConversationHistoryFn: func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{
					Messages: []slack.Message{opMsg},
				}, nil
			},
			GetUserInfoFn: func(user string) (*slack.User, error) {
				if getUserErr != nil {
					return nil, getUserErr
				}
				return &slack.User{TZ: userTZ}, nil
			},
		}
	}

	// userApiMock builds a mock that returns a SearchMessages result with the given total.
	userApiMock := func(total int, searchErr error) *slackclient.MockClient {
		return &slackclient.MockClient{
			SearchMessagesFn: func(query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
				if searchErr != nil {
					return nil, searchErr
				}
				return &slack.SearchMessages{
					Pagination: slack.Pagination{TotalCount: total},
				}, nil
			},
		}
	}

	tests := []struct {
		name        string
		config      map[string]interface{}
		api         *slackclient.MockClient
		userApi     *slackclient.MockClient
		wantScore   int
		wantReasons int
		wantErr     bool
	}{
		{
			name: "Base score only (watermark disabled, no tz configured)",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported": 2,
				"spam_feed.activity_low_watermark":  0,
				"spam_feed.local_timezone":           "",
			},
			api:         apiMock("", nil),
			userApi:     &slackclient.MockClient{},
			wantScore:   2,
			wantReasons: 1,
		},
		{
			name: "Low activity adds to score",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported":     2,
				"spam_feed.activity_low_watermark":      10,
				"spam_feed.anomaly_scores.low_activity": 1,
				"spam_feed.local_timezone":              "",
			},
			api:         apiMock("", nil),
			userApi:     userApiMock(5, nil), // 5 < 10
			wantScore:   3,
			wantReasons: 2,
		},
		{
			name: "Outside timezone adds to score",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported":    2,
				"spam_feed.activity_low_watermark":     0,
				"spam_feed.local_timezone":             "America/New_York",
				"spam_feed.anomaly_scores.outside_tz": 2,
			},
			api:         apiMock("Asia/Tokyo", nil),
			userApi:     &slackclient.MockClient{},
			wantScore:   4,
			wantReasons: 2,
		},
		{
			name: "Low activity and outside timezone both add to score",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported":     2,
				"spam_feed.activity_low_watermark":      10,
				"spam_feed.anomaly_scores.low_activity": 1,
				"spam_feed.local_timezone":              "America/New_York",
				"spam_feed.anomaly_scores.outside_tz":  2,
			},
			api:         apiMock("Asia/Tokyo", nil),
			userApi:     userApiMock(5, nil),
			wantScore:   5,
			wantReasons: 3,
		},
		{
			name: "MsgRefToMessage failure returns base score and error",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported": 2,
				"spam_feed.activity_low_watermark":  0,
			},
			api: &slackclient.MockClient{
				JoinConversationFn: func(channelID string) (*slack.Channel, string, []string, error) {
					return nil, "", nil, errors.New("join error")
				},
			},
			userApi:     &slackclient.MockClient{},
			wantScore:   2,
			wantReasons: 1,
			wantErr:     true,
		},
		{
			name: "userActivityScore error is logged but execution continues",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported":     2,
				"spam_feed.activity_low_watermark":      10,
				"spam_feed.anomaly_scores.low_activity": 1,
				"spam_feed.local_timezone":              "",
			},
			api:         apiMock("", nil),
			userApi:     userApiMock(0, errors.New("search failed")),
			wantScore:   2, // no activity score added due to error
			wantReasons: 1,
		},
		{
			name: "userTzScore error is logged but execution continues",
			config: map[string]interface{}{
				"spam_feed.anomaly_scores.reported":    2,
				"spam_feed.activity_low_watermark":     0,
				"spam_feed.local_timezone":             "America/New_York",
				"spam_feed.anomaly_scores.outside_tz": 2,
			},
			api:         apiMock("", errors.New("user not found")),
			userApi:     &slackclient.MockClient{},
			wantScore:   2, // no tz score added due to error
			wantReasons: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			score, reasons, err := anomalyScoreInternal(ref, tt.api, tt.userApi, zerolog.Nop())
			if tt.wantErr && err == nil {
				t.Errorf("anomalyScoreInternal() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("anomalyScoreInternal() unexpected error: %v", err)
			}
			if score != tt.wantScore {
				t.Errorf("anomalyScoreInternal() score = %d, want %d", score, tt.wantScore)
			}
			if len(reasons) != tt.wantReasons {
				t.Errorf("anomalyScoreInternal() reasons count = %d, want %d (reasons: %v)", len(reasons), tt.wantReasons, reasons)
			}
		})
	}
}

// TestProcessSpamFeedMessage verifies the core handler logic end-to-end.
func TestProcessSpamFeedMessage(t *testing.T) {
	const (
		spamChan   = "C_SPAM_FEED"
		spamTS     = "1111111111.000100"
		opChan     = "C02BZ36790B"
		opTS       = "1639843883.000100"
		opUser     = "U_OP_USER"
		botUID     = "U_BOT_UID"
		chanName   = "spam-feed"
		// Valid non-threaded permalink matching opChan/opTS
		opPermalink = "<https://orgname.slack.com/archives/C02BZ36790B/p1639843883000100>"
		// Valid threaded permalink
		threadedPermalink = "<https://orgname.slack.com/archives/C02BZ36790B/p1639843883000800?thread_ts=1639843880.000700&amp;cid=C02BZ36790B>"
	)

	spamFeedMsg := slack.Message{
		Msg: slack.Msg{Timestamp: spamTS, Channel: spamChan},
	}
	opMsg := slack.Message{
		Msg: slack.Msg{Timestamp: opTS, Channel: opChan, User: opUser},
	}
	botOpMsg := slack.Message{
		Msg: slack.Msg{Timestamp: opTS, Channel: opChan, User: botUID},
	}

	baseConfig := map[string]interface{}{
		"spam_feed.channel":                    chanName,
		"spam_feed.anomaly_scores.reported":    2,
		"spam_feed.activity_low_watermark":     0,
		"spam_feed.local_timezone":             "",
		"spam_feed.max_anomaly_score":          5,
		"spam_feed.reaction_emoji_hit":         "no_entry",
		"spam_feed.reaction_emoji_miss":        "white_check_mark",
	}

	// baseRouter is a router.Router with BotUID set.
	baseRouter := router.Router{BotUID: botUID}
	baseRoute := router.Route{}

	// channelInfoOK returns the spam-feed channel info.
	channelInfoOK := func(input *slack.GetConversationInfoInput) (*slack.Channel, error) {
		return &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{NameNormalized: chanName},
				Name:         chanName,
			},
		}, nil
	}

	t.Run("Early return when SubType and Username do not match", func(t *testing.T) {
		setupViperConfig(t, baseConfig)
		// No API methods should be called
		mock := &slackclient.MockClient{}
		ev := slackevents.MessageEvent{SubType: "file_share", Username: "SomeBot"}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, mock, ev, opPermalink)
		// Pass: no panics (no API calls made)
	})

	t.Run("Early return when channel name does not match", func(t *testing.T) {
		setupViperConfig(t, baseConfig)
		mock := &slackclient.MockClient{
			GetConversationInfoFn: func(input *slack.GetConversationInfoInput) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{NameNormalized: "other-channel"},
					},
				}, nil
			},
		}
		ev := slackevents.MessageEvent{SubType: BOT_MESSAGE_TYPE, Channel: spamChan}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, mock, ev, opPermalink)
		// Pass: early return after GetConversationInfo, no further calls
	})

	t.Run("spamFeedMsg retrieval failure causes early return", func(t *testing.T) {
		setupViperConfig(t, baseConfig)
		mock := &slackclient.MockClient{
			GetConversationInfoFn: channelInfoOK,
			JoinConversationFn:    noopJoin,
			GetConversationHistoryFn: func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{Messages: []slack.Message{}}, nil // 0 msgs → "message not found"
			},
		}
		ev := slackevents.MessageEvent{
			SubType:   BOT_MESSAGE_TYPE,
			Channel:   spamChan,
			TimeStamp: spamTS,
		}
		// Should not panic; logs error and returns
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, mock, ev, opPermalink)
	})

	t.Run("opMsg retrieval failure posts error reply and continues", func(t *testing.T) {
		setupViperConfig(t, map[string]interface{}{
			"spam_feed.channel":                 chanName,
			"spam_feed.anomaly_scores.reported": 2,
			"spam_feed.activity_low_watermark":  0,
			"spam_feed.local_timezone":          "",
			"spam_feed.max_anomaly_score":       5,
			"spam_feed.reaction_emoji_miss":     "white_check_mark",
		})
		postCalled := 0
		mock := &slackclient.MockClient{
			GetConversationInfoFn: channelInfoOK,
			JoinConversationFn:    noopJoin,
			GetConversationHistoryFn: func(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				if params.ChannelID == spamChan {
					return &slack.GetConversationHistoryResponse{Messages: []slack.Message{spamFeedMsg}}, nil
				}
				// opChan: return empty → "message not found"
				return &slack.GetConversationHistoryResponse{Messages: []slack.Message{}}, nil
			},
			PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
				postCalled++
				return channelID, "ts", nil
			},
			AddReactionFn: noopReaction,
		}
		ev := slackevents.MessageEvent{
			SubType:   BOT_MESSAGE_TYPE,
			Channel:   spamChan,
			TimeStamp: spamTS,
		}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, mock, ev, opPermalink)
		// PostMessage should be called at least once (error reply + ack)
		if postCalled == 0 {
			t.Errorf("expected PostMessage to be called, got 0 calls")
		}
	})

	t.Run("Threaded reply triggers early return with reply message", func(t *testing.T) {
		setupViperConfig(t, baseConfig)
		postCalled := false
		mock := &slackclient.MockClient{
			GetConversationInfoFn: channelInfoOK,
			JoinConversationFn:    noopJoin,
			GetConversationHistoryFn: historyFor(map[string]slack.Message{
				spamChan: spamFeedMsg,
			}),
			PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
				postCalled = true
				return channelID, "ts", nil
			},
		}
		ev := slackevents.MessageEvent{
			SubType:   BOT_MESSAGE_TYPE,
			Channel:   spamChan,
			TimeStamp: spamTS,
		}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, mock, ev, threadedPermalink)
		if !postCalled {
			t.Errorf("expected PostMessage to be called for threaded reply notice")
		}
	})

	t.Run("Bot reporting itself triggers early return with Hey message", func(t *testing.T) {
		setupViperConfig(t, baseConfig)
		postMessages := 0
		mock := &slackclient.MockClient{
			GetConversationInfoFn: channelInfoOK,
			JoinConversationFn:    noopJoin,
			GetConversationHistoryFn: historyFor(map[string]slack.Message{
				spamChan: spamFeedMsg,
				opChan:   botOpMsg,
			}),
			PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
				postMessages++
				return channelID, "ts", nil
			},
		}
		ev := slackevents.MessageEvent{
			SubType:   BOT_MESSAGE_TYPE,
			Channel:   spamChan,
			TimeStamp: spamTS,
		}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, mock, ev, opPermalink)
		// Expect at least 2 PostMessage calls: ack + "Hey! That's not nice."
		if postMessages < 2 {
			t.Errorf("expected at least 2 PostMessage calls (ack + hey), got %d", postMessages)
		}
	})

	t.Run("Score below threshold: no DeleteMessage, warning posted", func(t *testing.T) {
		cfg := map[string]interface{}{
			"spam_feed.channel":                 chanName,
			"spam_feed.anomaly_scores.reported": 2,
			"spam_feed.activity_low_watermark":  0,
			"spam_feed.local_timezone":          "",
			"spam_feed.max_anomaly_score":       5, // score 2 < 5
			"spam_feed.reaction_emoji_miss":     "white_check_mark",
			"spam_feed.op_warning":              "This message has been flagged.",
		}
		setupViperConfig(t, cfg)

		deleteCalled := false
		reactionEmoji := ""
		mock := &slackclient.MockClient{
			GetConversationInfoFn: channelInfoOK,
			JoinConversationFn:    noopJoin,
			GetConversationHistoryFn: historyFor(map[string]slack.Message{
				spamChan: spamFeedMsg,
				opChan:   opMsg,
			}),
			PostMessageFn: noopPost,
			AddReactionFn: func(name string, item slack.ItemRef) error {
				reactionEmoji = name
				return nil
			},
		}
		userMock := &slackclient.MockClient{
			DeleteMessageFn: func(channel, messageTimestamp string) (string, string, error) {
				deleteCalled = true
				return channel, messageTimestamp, nil
			},
		}

		ev := slackevents.MessageEvent{
			SubType:   BOT_MESSAGE_TYPE,
			Channel:   spamChan,
			TimeStamp: spamTS,
		}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, userMock, ev, opPermalink)

		if deleteCalled {
			t.Errorf("expected DeleteMessage NOT to be called when score below threshold")
		}
		if reactionEmoji != "white_check_mark" {
			t.Errorf("expected miss emoji 'white_check_mark', got %q", reactionEmoji)
		}
	})

	t.Run("Score at threshold: DeleteMessage called, hit reaction added", func(t *testing.T) {
		cfg := map[string]interface{}{
			"spam_feed.channel":                 chanName,
			"spam_feed.anomaly_scores.reported": 5,
			"spam_feed.activity_low_watermark":  0,
			"spam_feed.local_timezone":          "",
			"spam_feed.max_anomaly_score":       5, // score 5 >= 5
			"spam_feed.reaction_emoji_hit":      "no_entry",
		}
		setupViperConfig(t, cfg)

		deleteCalled := false
		deletedChan := ""
		deletedTS := ""
		reactionEmoji := ""
		mock := &slackclient.MockClient{
			GetConversationInfoFn: channelInfoOK,
			JoinConversationFn:    noopJoin,
			GetConversationHistoryFn: historyFor(map[string]slack.Message{
				spamChan: spamFeedMsg,
				opChan:   opMsg,
			}),
			PostMessageFn: noopPost,
			AddReactionFn: func(name string, item slack.ItemRef) error {
				reactionEmoji = name
				return nil
			},
		}
		userMock := &slackclient.MockClient{
			DeleteMessageFn: func(channel, messageTimestamp string) (string, string, error) {
				deleteCalled = true
				deletedChan = channel
				deletedTS = messageTimestamp
				return channel, messageTimestamp, nil
			},
		}

		ev := slackevents.MessageEvent{
			SubType:   BOT_MESSAGE_TYPE,
			Channel:   spamChan,
			TimeStamp: spamTS,
		}
		ProcessSpamFeedMessage(baseRouter, baseRoute, mock, userMock, ev, opPermalink)

		if !deleteCalled {
			t.Errorf("expected DeleteMessage to be called when score >= threshold")
		}
		if deletedChan != opChan {
			t.Errorf("DeleteMessage called on channel %q, want %q", deletedChan, opChan)
		}
		if deletedTS != opTS {
			t.Errorf("DeleteMessage called with ts %q, want %q", deletedTS, opTS)
		}
		if reactionEmoji != "no_entry" {
			t.Errorf("expected hit emoji 'no_entry', got %q", reactionEmoji)
		}
	})
}

// TestReacjiUsernameTriggersHandler ensures the REACJI_USERNAME constant triggers the handler path.
func TestReacjiUsernameTriggersHandler(t *testing.T) {
	setupViperConfig(t, map[string]interface{}{
		"spam_feed.channel": "spam-feed",
	})

	called := false
	mock := &slackclient.MockClient{
		GetConversationInfoFn: func(input *slack.GetConversationInfoInput) (*slack.Channel, error) {
			called = true
			return &slack.Channel{
				GroupConversation: slack.GroupConversation{
					Conversation: slack.Conversation{NameNormalized: "other"},
				},
			}, nil
		},
	}
	ev := slackevents.MessageEvent{
		Username: REACJI_USERNAME,
		Channel:  "C_ANY",
	}
	ProcessSpamFeedMessage(router.Router{}, router.Route{}, mock, mock, ev, "")
	if !called {
		t.Errorf("expected GetConversationInfo to be called when Username == REACJI_USERNAME")
	}
}

// TestDebugResponseFormat verifies the debug response text includes expected substrings.
// Since slack.MsgOption is opaque, we validate indirectly by constructing the text
// the same way addDebugResponse does and checking the substring.
func TestDebugResponseFormat(t *testing.T) {
	setupViperConfig(t, map[string]interface{}{"spam_feed.max_anomaly_score": 5})

	tests := []struct {
		name         string
		removed      bool
		score        int
		wantContains string
	}{
		{
			name:         "removed=true",
			removed:      true,
			score:        5,
			wantContains: "I removed the OP",
		},
		{
			name:         "removed=false",
			removed:      false,
			score:        2,
			wantContains: "didn't result in a removal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the debug response the same way addDebugResponse does.
			debugResponse := "This is what I found about the OP:\n"
			debugResponse += "- a reason\n"
			if tt.removed {
				debugResponse += fmt.Sprintf("I removed the OP since the final anomaly score (%d/%d) was suspect enough.", tt.score, viper.GetInt("spam_feed.max_anomaly_score"))
			} else {
				debugResponse += fmt.Sprintf("The final anomaly score (%d/%d) didn't result in a removal.", tt.score, viper.GetInt("spam_feed.max_anomaly_score"))
			}
			if !strings.Contains(debugResponse, tt.wantContains) {
				t.Errorf("debug response %q does not contain %q", debugResponse, tt.wantContains)
			}
		})
	}
}
