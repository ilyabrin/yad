package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// TODO: add Clean() cmd
// TODO: add filter for fields will print in table
var trashCmd = &cobra.Command{
	Use: "trash",
}

var listTrashCmd = &cobra.Command{
	Use: "list",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Trash.List(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		// TODO: prettify
		fmt.Println(resp)

	},
}

var restoreTrashCmd = &cobra.Command{
	Use: "trash",
	Run: func(cmd *cobra.Command, args []string) {
		resp, oper, err := yadisk.Trash.Restore(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		// TODO: prettify
		fmt.Println(oper.Status)
		fmt.Println(resp)

	},
}

var deleteTrashCmd = &cobra.Command{
	Use: "list",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Trash.Delete(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		// TODO: prettify
		fmt.Println(resp)

	},
}

func init() {
	rootCmd.AddCommand(trashCmd)
	trashCmd.AddCommand(
		listTrashCmd,
		restoreTrashCmd,
		deleteTrashCmd,
	)
}
