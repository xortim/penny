package help

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func setupViperConfig(t *testing.T, cfg map[string]interface{}) {
	for k, v := range cfg {
		viper.Set(k, v)
	}
	t.Cleanup(viper.Reset)
}

func TestFormatHelp(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "default config shows emoji, no assistance channel",
			config: map[string]interface{}{
				"spam_feed.emoji": "no_entry_sign",
			},
			wantContains: []string{
				"no_entry_sign",
				"Penny",
			},
			wantAbsent: []string{
				"<#",
			},
		},
		{
			name: "with assistance channel configured",
			config: map[string]interface{}{
				"spam_feed.emoji":                 "no_entry_sign",
				"spam_feed.assistance_channel_id": "C12345ABC",
			},
			wantContains: []string{
				"no_entry_sign",
				"<#C12345ABC>",
			},
		},
		{
			name: "custom emoji",
			config: map[string]interface{}{
				"spam_feed.emoji": "rotating_light",
			},
			wantContains: []string{
				"rotating_light",
			},
			wantAbsent: []string{
				"no_entry_sign",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupViperConfig(t, tt.config)

			got := formatHelp()

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("formatHelp() missing %q in:\n%s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("formatHelp() should not contain %q in:\n%s", absent, got)
				}
			}
		})
	}
}

func TestGetSlashCommandRoutes(t *testing.T) {
	setupViperConfig(t, map[string]interface{}{
		"spam_feed.emoji": "no_entry_sign",
	})

	routes := GetSlashCommandRoutes()
	if len(routes) != 1 {
		t.Fatalf("GetSlashCommandRoutes() returned %d routes, want 1", len(routes))
	}

	route := routes[0]
	if route.Command != "/help" {
		t.Errorf("route Command = %q, want %q", route.Command, "/help")
	}
	if route.Name != "help.help" {
		t.Errorf("route Name = %q, want %q", route.Name, "help.help")
	}
	if route.ImmediateResponse == "" {
		t.Error("route ImmediateResponse is empty")
	}
	if route.Plugin == nil {
		t.Error("route Plugin is nil")
	}
}
