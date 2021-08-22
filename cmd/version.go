package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xortim/bones/conf"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Long:  `Show version`,
		Run:   printVersion,
	}
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println(conf.Executable + " - " + conf.GitVersion)
}
