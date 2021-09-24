package hallmonitor

import (
	"github.com/gadget-bot/gadget/router"
)

func GetChannelMessageRoutes() []router.ChannelMessageRoute {
	return []router.ChannelMessageRoute{
		*monitorSpamFeedMessages(),
	}
}
