package whatsnew

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/xortim/penny/pkg/slackclient"
)

const testChangelog = `# Changelog

## [Unreleased]

### Added

- New unreleased feature

## [0.2] - 2026-02-24

### Added

- Add "what's new?" mention command (#14)

## [0.1] - 2026-02-21

### Changed

- Update main (#33)
`

func TestFormatWhatsNew(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		raw          string
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "bare what's new returns latest",
			message:      "what's new",
			raw:          testChangelog,
			wantContains: []string{"Latest Changes", "New unreleased feature"},
		},
		{
			name:         "what's new? with question mark",
			message:      "what's new?",
			raw:          testChangelog,
			wantContains: []string{"Latest Changes"},
		},
		{
			name:         "whats new without apostrophe",
			message:      "whats new",
			raw:          testChangelog,
			wantContains: []string{"Latest Changes"},
		},
		{
			name:         "since version returns sections after that version",
			message:      "what's new since 0.1",
			raw:          testChangelog,
			wantContains: []string{"Latest Changes", "0.2"},
		},
		{
			name:         "since version with v prefix",
			message:      "what's new since v0.1",
			raw:          testChangelog,
			wantContains: []string{"Latest Changes", "0.2"},
		},
		{
			name:         "since unknown version returns error",
			message:      "what's new since 9.9",
			raw:          testChangelog,
			wantErr:      true,
		},
		{
			name:    "empty changelog returns error",
			message: "what's new",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatWhatsNew(tt.message, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Errorf("formatWhatsNew() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("formatWhatsNew() unexpected error: %v", err)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("formatWhatsNew() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

func TestProcessWhatsNew(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		raw         string
		wantPost    bool
		wantChannel string
	}{
		{
			name:        "successful response posts to channel",
			message:     "what's new",
			raw:         testChangelog,
			wantPost:    true,
			wantChannel: "C123",
		},
		{
			name:        "error response still posts to channel",
			message:     "what's new",
			raw:         "",
			wantPost:    true,
			wantChannel: "C123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postCalled := false
			postedChannel := ""
			mock := &slackclient.MockClient{
				PostMessageFn: func(channelID string, options ...slack.MsgOption) (string, string, error) {
					postCalled = true
					postedChannel = channelID
					return channelID, "ts", nil
				},
			}

			ev := slackevents.AppMentionEvent{
				Channel:   "C123",
				TimeStamp: "1234567890.000100",
			}

			processWhatsNew(mock, ev, tt.message, tt.raw)

			if postCalled != tt.wantPost {
				t.Errorf("PostMessage called = %v, want %v", postCalled, tt.wantPost)
			}
			if tt.wantPost && postedChannel != tt.wantChannel {
				t.Errorf("PostMessage channel = %q, want %q", postedChannel, tt.wantChannel)
			}
		})
	}
}

func TestGetMentionRoutes(t *testing.T) {
	routes := GetMentionRoutes(testChangelog)
	if len(routes) != 1 {
		t.Fatalf("GetMentionRoutes() returned %d routes, want 1", len(routes))
	}
	if routes[0].Name != "whatsnew.whatsNew" {
		t.Errorf("route name = %q, want %q", routes[0].Name, "whatsnew.whatsNew")
	}
	if routes[0].Plugin == nil {
		t.Error("route Plugin is nil")
	}
}
