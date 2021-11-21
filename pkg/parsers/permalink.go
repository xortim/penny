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

	if len(pathParts) != 0 {
		ref.Channel = pathParts[2]
		ref.Timestamp = PermalinkPathTS(pathParts[3])
	}

	return *ref, query.Has("thread_ts")
}

// PremalinkPathTS expects the string timestamp representation from a permalink.
// this is the Timestamp but without any decimal and prefixed with p
func PermalinkPathTS(str string) string {
	str = strings.TrimPrefix(str, "p")
	l := len(str)
	// the ts format has 6 digits after the deciminal.
	return fmt.Sprintf("%s.%s", str[:l-6], str[l-6:l])
}
