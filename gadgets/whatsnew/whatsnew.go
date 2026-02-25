package whatsnew

import (
	"regexp"
	"strings"

	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/xortim/penny/pkg/changelog"
	"github.com/xortim/penny/pkg/slackclient"
)

var sinceRe = regexp.MustCompile(`(?i)what'?s\s+new\s+since\s+v?(\S+)`)

// GetMentionRoutes returns the mention routes for the whatsnew gadget.
func GetMentionRoutes(raw string) []router.MentionRoute {
	return []router.MentionRoute{
		{
			Route: router.Route{
				Name:        "whatsnew.whatsNew",
				Pattern:     `(?i)what'?s\s+new`,
				Description: "Show recent changelog entries",
				Help:        "what's new [since <version>]",
			},
			Plugin: func(r router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
				processWhatsNew(&api, ev, message, raw)
			},
		},
	}
}

// processWhatsNew contains the testable core logic.
func processWhatsNew(api slackclient.Client, ev slackevents.AppMentionEvent, message string, raw string) {
	logger := log.With().Str("channel", ev.Channel).Str("user", ev.User).Logger()
	logger.Debug().Str("message", message).Msg("processing what's new request")

	text, err := formatWhatsNew(message, raw)
	if err != nil {
		logger.Error().Err(err).Msg("failed to format changelog")
		text = "Sorry, I couldn't retrieve the changelog."
	}

	_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(text, false), slack.MsgOptionTS(ev.TimeStamp))
	if err != nil {
		logger.Error().Err(err).Msg("failed to post what's new reply")
	}
}

// formatWhatsNew is a pure function that formats the changelog response.
func formatWhatsNew(message string, raw string) (string, error) {
	cl := changelog.Parse(raw)

	if m := sinceRe.FindStringSubmatch(message); m != nil {
		version := strings.TrimPrefix(m[1], "v")
		return cl.Since(version)
	}

	return cl.Latest()
}
