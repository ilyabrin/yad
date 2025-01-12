package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "A brief description of your command",
	Long:  `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Resources.Publish(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(resp.Href)
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
