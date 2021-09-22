package parsers

import (
	"fmt"
	"regexp"

	"github.com/slack-go/slack"
)

var (
	chatPermaLink = regexp.MustCompile(`^<?https://[^/]+/archives/(?P<channel>[^/]+)/p(?P<ts>[^/>]+)>?$`)
)

// ChatPermalinkToMsgRef converts a permalink to slack.ItemRef
func NewRefToMessageFromPermalink(url string) slack.ItemRef {
	ref := &slack.ItemRef{}

	matches := chatPermaLink.FindStringSubmatch(url)

	if len(matches) != 0 {
		l := len(matches[2])
		ref.Channel = matches[1]
		ref.Timestamp = fmt.Sprintf("%s.%s", matches[2][:l-6], matches[2][l-6:l])
	}

	return *ref
}
