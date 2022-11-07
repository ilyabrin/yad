package cmd

import (
	"context"
	"log"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// TODO: add filter for fields will print in table
// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Add info information to resource",
	Long:  `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Disk.Info(context.Background(), nil)
		if err != nil {
			log.Println(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		t.AppendHeader(table.Row{
			"used_login", // TODO: move to meCmd
			"user_id",
			"max_file_size",
			// TODO: add field in Disk package
			// "paid_max_file_size",
			"total_space",
			"trash_size",
			"is_paid",
			"used_space",
			"unlimited_autoupload_enabled",
			// TODO: add system_folders list (systemFoldersCmd)
		})

		t.AppendRows([]table.Row{
			{
				resp.User.Login,
				resp.User.UID,
				humanize.Bytes(uint64(resp.MaxFileSize)),
				// TODO: add field in Disk package
				// resp.PaidMaxFileSize,
				humanize.Bytes(uint64(resp.TotalSpace)),
				humanize.Bytes(uint64(resp.TrashSize)),
				resp.IsPaid,
				humanize.Bytes(uint64(resp.UsedSpace)),
				resp.UnlimitedAutouploadEnabled,
				// TODO: add system_folders list (systemFoldersCmd)
			},
		})
		t.AppendSeparator()
		// t.SetStyle(table.StyleColoredCyanWhiteOnBlack)
		t.Render()

	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
