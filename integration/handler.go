package integration

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/gadget-bot/gadget/router"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/xortim/penny/gadgets/hallmonitor"
)

// TestHandler replicates Gadget's MessageEvent dispatch for integration testing.
// It verifies the Slack signature, parses the event, matches a route, and
// executes the handler — all without requiring a database or gadget.Run().
type TestHandler struct {
	SigningSecret string
	BotUID       string
	APIClient    *slack.Client // bot token client pointed at mock server
	UserClient   *slack.Client // user token client pointed at mock server
}

// ServeHTTP implements http.Handler, replicating Gadget's /gadget endpoint.
func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify Slack request signature
	sv, err := slack.NewSecretsVerifier(r.Header, h.SigningSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse the Slack event
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if eventsAPIEvent.Type != slackevents.CallbackEvent {
		w.WriteHeader(http.StatusOK)
		return
	}

	innerEvent := eventsAPIEvent.InnerEvent
	ev, ok := innerEvent.Data.(*slackevents.MessageEvent)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Build the router (no DB needed for MessageEvent dispatch)
	rtr := router.Router{BotUID: h.BotUID}

	// Get hallmonitor's registered routes
	routes := hallmonitor.GetChannelMessageRoutes()

	// Find a matching route by regex pattern (same as Gadget's FindChannelMessageRouteByMessage)
	message := ev.Text
	var matchedRoute *router.ChannelMessageRoute
	for i, route := range routes {
		re := regexp.MustCompile(route.Pattern)
		if re.MatchString(message) {
			matchedRoute = &routes[i]
			break
		}
	}

	if matchedRoute == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Execute synchronously (unlike Gadget's async goroutine) so tests can assert after return.
	// Call ProcessSpamFeedMessage directly with both clients pointed at the mock server,
	// bypassing handleSpamFeedMessage which creates its own userApi from viper config.
	hallmonitor.ProcessSpamFeedMessage(rtr, matchedRoute.Route, h.APIClient, h.UserClient, *ev, message)

	w.WriteHeader(http.StatusOK)
}
