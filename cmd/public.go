package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// TODO: add filter for fields will print in table
// publicCmd represents the info command
var publicCmd = &cobra.Command{
	Use:     "public",
	Aliases: []string{"pub", "p"},
}

var publicInfoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"i"},
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: rename Meta() to Info()
		resp, err := yadisk.Public.Meta(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		// TODO: add all fields
		t.AppendHeader(table.Row{
			"owner",
			"name",
			"type",
			"views_count",
			"created",
			"updated",
		})

		t.AppendRows([]table.Row{
			{
				resp.Owner.DisplayName,
				resp.Name,
				resp.Type,
				resp.ViewsCount,
				resp.Created,
				resp.Modified,
			},
		})
		t.AppendSeparator()
		t.Render()

	},
}

// TODO: move to Resources
var listPublicCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Resources.ListPublic(context.Background(), nil)
		if err != nil {
			log.Println(err)
		}

		// TODO: prettify
		fmt.Println(resp)
	},
}

var savePublicCmd = &cobra.Command{
	Use:     "save",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Public.Save(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		// TODO: prettify
		println("saved:", resp.Href)

	},
}

var downloadPublicCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"url", "d"},
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Public.DownloadURL(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}
		// TODO: prettify
		println(resp.Href)
	},
}

// yad public [ list (default) | meta | save | download [url] ]

func init() {
	rootCmd.AddCommand(publicCmd)
	publicCmd.AddCommand(publicInfoCmd, listPublicCmd, savePublicCmd, downloadPublicCmd)
}
