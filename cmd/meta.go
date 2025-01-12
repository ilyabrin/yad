package cmd

import (
	"context"
	"log"
	"os"

	"github.com/ilyabrin/disk"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// TODO: move table render's code to callback for clean code

var setCmd = &cobra.Command{
	Use:     "set",
	Aliases: []string{"s"},
	Short:   "Set key and value",
	Long:    "Set key and value",
	Args:    cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {

		path := args[0]
		key := args[1]
		val := args[2]

		meta := disk.Metadata{
			"custom_properties": {
				key: &val,
			},
		}
		_, err := yadisk.Resources.UpdateMeta(context.Background(), path, &meta)
		if err != nil {
			log.Println(err)
		}
	},
}

var getCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"g"},
	Short:   "Get resource meta keys",
	Long:    "Get resource meta keys",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := yadisk.Resources.Meta(context.Background(), args[0], nil)
		if err != nil {
			log.Println(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"key_name", "key_value"})

		for key_name, key_value := range resp.CustomProperties {
			t.AppendRows([]table.Row{{key_name, key_value}})
		}

		t.SetAutoIndex(true)
		t.AppendSeparator()
		t.Render()
	},
}

// TODO: add purgeCmd() will delete all meta data for resource
var delCmd = &cobra.Command{
	Use:     "del",
	Aliases: []string{"d", "rm"},
	Short:   "Delete meta information key for resource",
	Long:    "Delete meta information key for resource",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		key := args[1]

		meta := disk.Metadata{"custom_properties": {
			key: nil,
		}}

		resp, err := yadisk.Resources.UpdateMeta(context.Background(), path, &meta)
		if err != nil {
			log.Println(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		t.AppendHeader(table.Row{
			"#",
			"key_name",
			"key_value",
		})

		i := 1
		for k, v := range resp.CustomProperties {
			t.AppendRows([]table.Row{
				{
					i,
					k,
					v,
				},
			})
			i++
		}

		t.AppendSeparator()
		t.Render()
	},
}

var metaCmd = &cobra.Command{
	Use:     "meta",
	Aliases: []string{"m"},
	Short:   "Get meta information of resource",
	Long:    `A longer description that spans multiple lines`,
}

func init() {
	rootCmd.AddCommand(metaCmd)
	metaCmd.AddCommand(setCmd, getCmd, delCmd)
}
