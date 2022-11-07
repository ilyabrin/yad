package cmd

import (
	"github.com/spf13/cobra"
)

// metaCmd represents the meta command
var metaCmd = &cobra.Command{
	Use:     "meta",
	Aliases: []string{"m"},
	Args:    cobra.ExactArgs(1),
	Short:   "Get meta information of resource",
	Long:    `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		println("not implemented")
	},
}

func init() {
	rootCmd.AddCommand(metaCmd)
}
