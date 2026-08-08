package cmd

import "testing"

func TestValidateSpamFeedConfig(t *testing.T) {
	tests := []struct {
		name        string
		channelName string
		userToken   string
		wantErr     bool
	}{
		{
			name:        "No spam-feed channel configured, no user token required",
			channelName: "",
			userToken:   "",
			wantErr:     false,
		},
		{
			name:        "Spam-feed channel configured with user token succeeds",
			channelName: "spam-feed",
			userToken:   "xoxp-token",
			wantErr:     false,
		},
		{
			name:        "Spam-feed channel configured without user token fails fast",
			channelName: "spam-feed",
			userToken:   "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpamFeedConfig(tt.channelName, tt.userToken)
			if tt.wantErr && err == nil {
				t.Errorf("validateSpamFeedConfig() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateSpamFeedConfig() unexpected error: %v", err)
			}
		})
	}
}
