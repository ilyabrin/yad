package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long:  `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Resources.CreateDir(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}

		fmt.Print(resp.Href)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
