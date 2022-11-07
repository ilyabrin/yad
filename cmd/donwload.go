package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"d"},
	Args:    cobra.ExactArgs(2),
	Short:   "Get link address for download resource",
	Long:    `A longer description that spans multiple lines`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: get download link for resource
		// TODO: download file to target dir = yad download what where
		// TODO: show progress
		remotePath := args[0]
		localPath := args[1]
		resp, err := yadisk.Resources.DownloadURL(context.Background(), remotePath, nil)
		if err != nil {
			log.Println(err)
		}

		downloadFile(resp.Href, localPath)
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}

// first naive implementation of file downloading
func downloadFile(url string, filepath string) (err error) {
	client := grab.NewClient()
	req, _ := grab.NewRequest(filepath, url)

	// start download
	fmt.Printf("Downloading %v...\n", req.URL())
	res := client.Do(req)
	fmt.Printf("  %v\n", res.HTTPResponse.Status)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			fmt.Printf(" downloaded %v from %v (%.2f%%)\n",
				humanize.Bytes(uint64(res.BytesComplete())),
				humanize.Bytes(uint64(res.Size())),
				100*res.Progress())

		case <-res.Done:
			// download is complete
			break Loop
		}
	}

	// check for errors
	if err := res.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Download saved to ./%v \n", res.Filename)

	// Output:
	// Downloading ...
	//   200 OK
	// 	downloaded 4.8 MB from 80 MB (5.92%)
	// 	downloaded 10 MB from 80 MB (12.95%)
	// 	downloaded 16 MB from 80 MB (20.14%)
	// 	downloaded 22 MB from 80 MB (27.05%)
	// 	downloaded 28 MB from 80 MB (34.27%)

	return nil
}
