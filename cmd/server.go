package cmd

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	myBot := gadget.Setup()

	// Plugin handlers go here
	myBot.Run()
	return nil
}

func setupServerFlags(c *cobra.Command) {
	c.PersistentFlags().IntP("port", "p", 3000, "The port on which the bot should bind.")
	viper.BindPFlag("server.port", c.PersistentFlags().Lookup("port"))
	viper.RegisterAlias("server.port", "listen.port")
	viper.SetDefault("server.port", 3000)

	c.PersistentFlags().String("db.hostname", "localhost", "The host for gadget's DB.")
	viper.BindPFlag("db.hostname", c.PersistentFlags().Lookup("db.hostname"))
	viper.RegisterAlias("db.hostname", "db.host")
	viper.SetDefault("db.hostname", "localhost")

	c.PersistentFlags().String("db.name", "gadget", "The name for gadget's DB.")
	viper.BindPFlag("db.name", c.PersistentFlags().Lookup("db.name"))
	viper.SetDefault("db.name", "gadget")

	c.PersistentFlags().String("db.username", "", "The username for gadget's DB.")
	viper.BindPFlag("db.username", c.PersistentFlags().Lookup("db.username"))
	viper.RegisterAlias("db.username", "db.user")
	viper.SetDefault("db.username", "gadget")

	c.PersistentFlags().String("db.password", "", "The password for gadget's DB.")
	viper.BindPFlag("db.password", c.PersistentFlags().Lookup("db.password"))
	viper.RegisterAlias("db.pass", "db.pass")
}
