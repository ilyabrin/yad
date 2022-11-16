package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// unpublishCmd represents the unpublish command
var unpublishCmd = &cobra.Command{
	Use:   "unpublish",
	Short: "A brief description of your command",
	Long:  `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Resources.Unpublish(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(resp.Href)
	},
}

func init() {
	rootCmd.AddCommand(unpublishCmd)
}
