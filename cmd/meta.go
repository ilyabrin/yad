package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// metaCmd represents the meta command
var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Add meta information to resource",
	Long:  `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("meta " + args[1] + " with value " + args[2] + " to next resource - " + args[3])
	},
}

func init() {
	rootCmd.AddCommand(metaCmd)
}
