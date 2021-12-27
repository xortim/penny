package cmd

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/gadget-bot/gadget/router"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xortim/penny/conf"
	"github.com/xortim/penny/gadgets/hallmonitor"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"serve"},
		Short:   "Run the bot",
		Long:    `Run the bot`,
		RunE:    server,
	}

	setupServerFlags(cmd)

	return cmd
}

func server(cmd *cobra.Command, args []string) error {
	myBot, err := gadget.SetupWithConfig(
		viper.GetString("slack.bot_oauth_token"),
		viper.GetString("slack.signing_secret"),
		viper.GetString("db.username"),
		viper.GetString("db.password"),
		viper.GetString("db.hostname"),
		viper.GetString("db.name"),
		viper.GetString("db.port"),
		viper.GetStringSlice("slack.global_admins"))
	if err != nil {
		return err
	}

	myBot.Router.ChannelMessageRoutes = make(map[string]router.ChannelMessageRoute)
	myBot.Router.AddChannelMessageRoutes(hallmonitor.GetChannelMessageRoutes())

	return myBot.Run()
}

func setupServerFlags(c *cobra.Command) {
	c.PersistentFlags().IntP("port", "p", 3000, "The port on which the bot should bind.")
	viper.BindPFlag("server.port", c.PersistentFlags().Lookup("port"))
	viper.RegisterAlias("listen.port", "server.port")
	viper.SetDefault("server.port", 3000)

	c.PersistentFlags().String("db_hostname", "localhost", "The host for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.hostname", c.PersistentFlags().Lookup("db_hostname"))
	viper.RegisterAlias("db.host", "db.hostname")
	viper.SetDefault("db.hostname", "localhost")

	c.PersistentFlags().String("db_name", conf.Executable, "The name for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.name", c.PersistentFlags().Lookup("db_name"))
	viper.SetDefault("db.name", conf.Executable)

	c.PersistentFlags().String("db_username", "", "The username for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.username", c.PersistentFlags().Lookup("db_username"))
	viper.RegisterAlias("db.user", "db.username")
	viper.SetDefault("db.username", conf.Executable)

	c.PersistentFlags().String("db_password", "", "The password for "+conf.Executable+"'s DB.")
	viper.BindPFlag("db.password", c.PersistentFlags().Lookup("db_password"))
	viper.RegisterAlias("db.pass", "db.password")
}
