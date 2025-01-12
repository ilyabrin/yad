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
			"paid_max_file_size",
			"total_space",
			"trash_size",
			"is_paid",
			"used_space",
			"unlimited_autoupload_enabled",
		})

		t.AppendRows([]table.Row{
			{
				resp.User.Login,
				resp.User.UID,
				humanize.Bytes(uint64(resp.MaxFileSize)),
				humanize.Bytes(uint64(resp.PaidMaxFileSize)),
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

var systemFoldersCmd = &cobra.Command{
	Use:   "sys",
	Short: "List all system folders",
	Long:  `List all system folders`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Disk.Info(context.Background(), nil)
		if err != nil {
			log.Println(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		// TODO: t.AppendHeader(table.Row{"path", "name", "size"})
		t.AppendHeader(table.Row{"path"})

		// TODO: eg. /social networks/ | google
		t.AppendRows([]table.Row{
			{resp.SystemFolders.Odnoklassniki},
			{resp.SystemFolders.Google},
			{resp.SystemFolders.Instagram},
			{resp.SystemFolders.Vkontakte},
			{resp.SystemFolders.Attach},
			{resp.SystemFolders.Mailru},
			{resp.SystemFolders.Downloads},
			{resp.SystemFolders.Applications},
			{resp.SystemFolders.Facebook},
			{resp.SystemFolders.Social},
			{resp.SystemFolders.Messenger},
			{resp.SystemFolders.Calendar},
			{resp.SystemFolders.Scans},
			{resp.SystemFolders.Screenshots},
			{resp.SystemFolders.Photostream},
		})

		t.SetAutoIndex(true)
		t.AppendSeparator()
		t.Render()
	},
}

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Get info about user",
	Long:  `Get info about user`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Disk.Info(context.Background(), nil)
		if err != nil {
			log.Println(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		t.AppendHeader(table.Row{
			"used_login",
			"user_id",
			"is_paid",
		})

		t.AppendRows([]table.Row{
			{
				resp.User.Login,
				resp.User.UID,
				resp.IsPaid,
			},
		})
		t.AppendSeparator()
		t.Render()

	},
}

func init() {
	rootCmd.AddCommand(infoCmd, systemFoldersCmd, meCmd)
}
