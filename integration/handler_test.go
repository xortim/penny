package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

const (
	testSigningSecret = "test-signing-secret-1234"
	testBotUID        = "U_BOT"
	testBotToken      = "xoxb-test-token"
	testUserToken     = "xoxp-test-user-token"

	spamFeedChan = "C_SPAM_FEED"
	spamFeedTS   = "1111111111.000100"
	spamFeedName = "spam-feed"

	opChan = "C_OP_CHAN"
	opTS   = "1639843883.000100"
	opUser = "U_SPAMMER"

	opPermalink = "https://orgname.slack.com/archives/C_OP_CHAN/p1639843883000100"
)

// setupViper configures Viper with test defaults and registers cleanup.
func setupViper(t *testing.T, overrides map[string]interface{}) {
	t.Helper()
	defaults := map[string]interface{}{
		"slack.user_oauth_token":               testUserToken,
		"spam_feed.channel":                    spamFeedName,
		"spam_feed.anomaly_scores.reported":    2,
		"spam_feed.anomaly_scores.low_activity": 1,
		"spam_feed.anomaly_scores.outside_tz":  2,
		"spam_feed.activity_low_watermark":     10,
		"spam_feed.local_timezone":             "America/New_York",
		"spam_feed.max_anomaly_score":          5,
		"spam_feed.reaction_emoji_hit":         "no_entry",
		"spam_feed.reaction_emoji_miss":        "white_check_mark",
		"spam_feed.emoji":                      "spam",
		"spam_feed.reacji_response":            "Looking into it.",
	}
	for k, v := range defaults {
		viper.Set(k, v)
	}
	for k, v := range overrides {
		viper.Set(k, v)
	}
	t.Cleanup(viper.Reset)
}

// buildEventPayload constructs a Slack event_callback JSON payload.
func buildEventPayload(t *testing.T, subType, username, channel, ts, text string) []byte {
	t.Helper()
	payload := map[string]interface{}{
		"token":    "XXYYZZ",
		"team_id":  "T1234",
		"type":     "event_callback",
		"event_id": "Ev1234",
		"event": map[string]interface{}{
			"type":     "message",
			"subtype":  subType,
			"username": username,
			"channel":  channel,
			"ts":       ts,
			"text":     "<" + text + ">",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal event payload: %v", err)
	}
	return body
}

// sendEvent sends a signed event payload to the test handler and returns the response.
func sendEvent(t *testing.T, handler http.Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/gadget", bytes.NewReader(body))
	SignRequest(req, body, testSigningSecret)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// newTestSetup creates a MockSlackServer, test handler, and properly configured slack clients.
func newTestSetup(t *testing.T, opts MockSlackOptions) (*MockSlackServer, *TestHandler) {
	t.Helper()
	mock := NewMockSlackServer(opts)
	t.Cleanup(mock.Close)

	apiClient := slack.New(testBotToken, slack.OptionAPIURL(mock.URL()))
	userClient := slack.New(testUserToken, slack.OptionAPIURL(mock.URL()))

	handler := &TestHandler{
		SigningSecret: testSigningSecret,
		BotUID:       testBotUID,
		APIClient:    apiClient,
		UserClient:   userClient,
	}

	return mock, handler
}

func TestSignatureVerification(t *testing.T) {
	_, handler := newTestSetup(t, MockSlackOptions{})
	setupViper(t, nil)

	t.Run("Valid signature returns 200", func(t *testing.T) {
		body := buildEventPayload(t, "bot_message", "Reacji Channeler", spamFeedChan, spamFeedTS, opPermalink)
		rec := sendEvent(t, handler, body)
		// May be 200 (route matched) or 404 (no match) but NOT 401
		if rec.Code == http.StatusUnauthorized {
			t.Errorf("expected valid signature to pass, got 401")
		}
	})

	t.Run("Invalid signature returns 401", func(t *testing.T) {
		body := buildEventPayload(t, "bot_message", "Reacji Channeler", spamFeedChan, spamFeedTS, opPermalink)
		req := httptest.NewRequest(http.MethodPost, "/gadget", bytes.NewReader(body))
		req.Header.Set("X-Slack-Request-Timestamp", "1234567890")
		req.Header.Set("X-Slack-Signature", "v0=bad_signature")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for invalid signature, got %d", rec.Code)
		}
	})
}

func TestIgnoredMessages(t *testing.T) {
	tests := []struct {
		name     string
		subType  string
		username string
	}{
		{
			name:     "Regular user message (no subtype, no reacji username)",
			subType:  "",
			username: "SomeUser",
		},
		{
			name:     "File share message",
			subType:  "file_share",
			username: "AnotherBot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, handler := newTestSetup(t, MockSlackOptions{
				ChannelName: spamFeedName,
			})
			setupViper(t, nil)

			body := buildEventPayload(t, tt.subType, tt.username, spamFeedChan, spamFeedTS, opPermalink)
			rec := sendEvent(t, handler, body)

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200 OK, got %d", rec.Code)
			}

			// The handler should return early before making any Slack API calls
			// beyond GetConversationInfo (which is called to check the channel name).
			// Specifically, no chat.postMessage calls should be made.
			postCalls := mock.CallsFor("chat.postMessage")
			if len(postCalls) > 0 {
				t.Errorf("expected no chat.postMessage calls for ignored message, got %d", len(postCalls))
			}
		})
	}
}

func TestSpamDetectionFullFlow(t *testing.T) {
	// Score: reported(2) + low_activity(1, since 5 < 10) + outside_tz(0, matching TZ) = 3
	// Threshold: 5 → no removal
	mock, handler := newTestSetup(t, MockSlackOptions{
		ChannelName: spamFeedName,
		HistoryMessages: map[string][]HistoryMessage{
			spamFeedChan: {{
				User:      "",
				Text:      "<" + opPermalink + ">",
				Timestamp: spamFeedTS,
			}},
			opChan: {{
				User:      opUser,
				Text:      "Buy my product!",
				Timestamp: opTS,
				Reactions: []HistoryReaction{{
					Name:  "spam",
					Users: []string{"U_REPORTER1"},
				}},
			}},
		},
		UserTZ:           "America/New_York", // matches local_timezone → tz score 0
		SearchTotalCount: 5,                  // below watermark 10 → low_activity score 1
	})
	setupViper(t, nil)

	body := buildEventPayload(t, "bot_message", "Reacji Channeler", spamFeedChan, spamFeedTS, opPermalink)
	rec := sendEvent(t, handler, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify conversations.join was called (bot joins channels to read messages)
	joinCalls := mock.CallsFor("conversations.join")
	if len(joinCalls) == 0 {
		t.Error("expected at least one conversations.join call")
	}

	// Verify conversations.history was called for both spam-feed and OP channels
	historyCalls := mock.CallsFor("conversations.history")
	if len(historyCalls) < 2 {
		t.Errorf("expected at least 2 conversations.history calls, got %d", len(historyCalls))
	}

	// Verify chat.postMessage was called (ack reply + debug response + OP warning)
	postCalls := mock.CallsFor("chat.postMessage")
	if len(postCalls) == 0 {
		t.Error("expected chat.postMessage calls for ack and debug response")
	}

	// Verify reactions.add was called with miss emoji (score 3 < threshold 5)
	reactionCalls := mock.CallsFor("reactions.add")
	if len(reactionCalls) == 0 {
		t.Error("expected reactions.add call")
	} else if reactionCalls[0].Params["name"] != "white_check_mark" {
		t.Errorf("expected miss emoji 'white_check_mark', got %q", reactionCalls[0].Params["name"])
	}

	// Verify NO chat.delete call (score below threshold)
	deleteCalls := mock.CallsFor("chat.delete")
	if len(deleteCalls) > 0 {
		t.Error("expected no chat.delete calls when score is below threshold")
	}

	// Verify users.info was called (for timezone check)
	userInfoCalls := mock.CallsFor("users.info")
	if len(userInfoCalls) == 0 {
		t.Error("expected users.info call for timezone scoring")
	}

	// Verify search.messages was called (for activity scoring)
	searchCalls := mock.CallsFor("search.messages")
	if len(searchCalls) == 0 {
		t.Error("expected search.messages call for activity scoring")
	}
}

func TestSpamRemovalFlow(t *testing.T) {
	// Score: reported(2) + low_activity(1, since 2 < 10) + outside_tz(2, different TZ) = 5
	// Threshold: 5 → removal triggered
	mock, handler := newTestSetup(t, MockSlackOptions{
		ChannelName: spamFeedName,
		HistoryMessages: map[string][]HistoryMessage{
			spamFeedChan: {{
				User:      "",
				Text:      "<" + opPermalink + ">",
				Timestamp: spamFeedTS,
			}},
			opChan: {{
				User:      opUser,
				Text:      "Crypto spam message",
				Timestamp: opTS,
			}},
		},
		UserTZ:           "Asia/Tokyo",  // different from America/New_York → tz score 2
		SearchTotalCount: 2,             // below watermark 10 → low_activity score 1
	})
	setupViper(t, nil)

	body := buildEventPayload(t, "bot_message", "Reacji Channeler", spamFeedChan, spamFeedTS, opPermalink)
	rec := sendEvent(t, handler, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify chat.delete was called (score >= threshold)
	deleteCalls := mock.CallsFor("chat.delete")
	if len(deleteCalls) == 0 {
		t.Error("expected chat.delete call when score meets threshold")
	} else {
		if deleteCalls[0].Params["channel"] != opChan {
			t.Errorf("chat.delete channel = %q, want %q", deleteCalls[0].Params["channel"], opChan)
		}
		if deleteCalls[0].Params["ts"] != opTS {
			t.Errorf("chat.delete ts = %q, want %q", deleteCalls[0].Params["ts"], opTS)
		}
	}

	// Verify reactions.add was called with hit emoji
	reactionCalls := mock.CallsFor("reactions.add")
	if len(reactionCalls) == 0 {
		t.Error("expected reactions.add call")
	} else if reactionCalls[0].Params["name"] != "no_entry" {
		t.Errorf("expected hit emoji 'no_entry', got %q", reactionCalls[0].Params["name"])
	}

	// Verify a removal warning was posted to the OP thread
	postCalls := mock.CallsFor("chat.postMessage")
	if len(postCalls) == 0 {
		t.Error("expected chat.postMessage calls")
	}
}

func TestScoreComponentsViaFullPipeline(t *testing.T) {
	tests := []struct {
		name             string
		userTZ           string
		searchTotal      int
		viperOverrides   map[string]interface{}
		expectDelete     bool
		expectReaction   string
	}{
		{
			name:           "Only reported score (2) — below threshold",
			userTZ:         "America/New_York", // matches → tz score 0
			searchTotal:    100,                // above watermark → activity score 0
			expectDelete:   false,
			expectReaction: "white_check_mark",
		},
		{
			name:           "Reported (2) + low activity (1) — below threshold",
			userTZ:         "America/New_York",
			searchTotal:    5,                  // below watermark → activity score 1
			expectDelete:   false,
			expectReaction: "white_check_mark",
		},
		{
			name:           "Reported (2) + outside TZ (2) — below threshold",
			userTZ:         "Asia/Tokyo",
			searchTotal:    100,
			expectDelete:   false,
			expectReaction: "white_check_mark",
		},
		{
			name:           "All components (2+1+2=5) — meets threshold",
			userTZ:         "Asia/Tokyo",
			searchTotal:    5,
			expectDelete:   true,
			expectReaction: "no_entry",
		},
		{
			name:        "High reported score alone meets threshold",
			userTZ:      "America/New_York",
			searchTotal: 100,
			viperOverrides: map[string]interface{}{
				"spam_feed.anomaly_scores.reported": 5,
			},
			expectDelete:   true,
			expectReaction: "no_entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, handler := newTestSetup(t, MockSlackOptions{
				ChannelName: spamFeedName,
				HistoryMessages: map[string][]HistoryMessage{
					spamFeedChan: {{
						Timestamp: spamFeedTS,
					}},
					opChan: {{
						User:      opUser,
						Text:      "spam",
						Timestamp: opTS,
					}},
				},
				UserTZ:           tt.userTZ,
				SearchTotalCount: tt.searchTotal,
			})
			setupViper(t, tt.viperOverrides)

			body := buildEventPayload(t, "bot_message", "Reacji Channeler", spamFeedChan, spamFeedTS, opPermalink)
			sendEvent(t, handler, body)

			deleteCalls := mock.CallsFor("chat.delete")
			if tt.expectDelete && len(deleteCalls) == 0 {
				t.Errorf("expected chat.delete call, got none")
			}
			if !tt.expectDelete && len(deleteCalls) > 0 {
				t.Errorf("expected no chat.delete calls, got %d", len(deleteCalls))
			}

			reactionCalls := mock.CallsFor("reactions.add")
			if len(reactionCalls) == 0 {
				t.Errorf("expected reactions.add call, got none")
			} else if reactionCalls[0].Params["name"] != tt.expectReaction {
				t.Errorf("reaction emoji = %q, want %q", reactionCalls[0].Params["name"], tt.expectReaction)
			}
		})
	}
}

func TestNonReacjiMessageIgnored(t *testing.T) {
	// A regular channel message (not from Reacji Channeler) should be ignored.
	mock, handler := newTestSetup(t, MockSlackOptions{
		ChannelName: spamFeedName,
	})
	setupViper(t, nil)

	// Build a payload with no subtype and regular username
	payload := map[string]interface{}{
		"token":    "XXYYZZ",
		"team_id":  "T1234",
		"type":     "event_callback",
		"event_id": "Ev5678",
		"event": map[string]interface{}{
			"type":    "message",
			"user":    "U_REGULAR",
			"channel": spamFeedChan,
			"ts":      "2222222222.000100",
			"text":    "Just a regular message",
		},
	}
	body, _ := json.Marshal(payload)

	rec := sendEvent(t, handler, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Should not make any chat.postMessage calls
	postCalls := mock.CallsFor("chat.postMessage")
	if len(postCalls) > 0 {
		t.Errorf("expected no chat.postMessage calls for regular message, got %d", len(postCalls))
	}
}
