package parsers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/slack-go/slack"
)

// NewRefToMessageFromPermalink converts a permalink to slack.ItemRef and determines if it is a threaded reply or not
func NewRefToMessageFromPermalink(str string) (slack.ItemRef, bool) {
	u, _ := url.Parse(str)
	pathParts := strings.Split(u.Path, "/")
	query, _ := url.ParseQuery(u.RawQuery)

	ref := &slack.ItemRef{}

	ts := PermalinkPathTS(pathParts[3])
	if len(pathParts) != 0 {
		ref.Channel = pathParts[2]
		ref.Timestamp = ts
	}

	isReply := query.Get("thread_ts") != "" && query.Get("thread_ts") != ts
	return *ref, isReply
}

// PremalinkPathTS expects the string timestamp representation from a permalink.
// this is the Timestamp but without any decimal and prefixed with p
func PermalinkPathTS(str string) string {
	str = strings.TrimPrefix(str, "p")
	l := len(str)
	// the ts format has 6 digits after the deciminal.
	return fmt.Sprintf("%s.%s", str[:l-6], str[l-6:l])
}
