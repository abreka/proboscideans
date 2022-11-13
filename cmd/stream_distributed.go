package cmd

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/abreka/proboscideans/accounts"

	"github.com/abreka/proboscideans/streaming"
	"github.com/spf13/cobra"
)

var streamDistributedCmd = &cobra.Command{
	Use:   "stream-distributed [credentials-dir]",
	Short: "stream events from multiple instances",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: make this not stupid
		// open a gzip file for writing.
		fp, err := os.Create(fmt.Sprintf("stream-%d.json.gz", time.Now().Unix()))
		if err != nil {
			cmd.PrintErrf("error opening file: %v", err)
			os.Exit(1)
		}
		defer fp.Close()
		gzWriter := gzip.NewWriter(fp)
		defer gzWriter.Close()

		ds, err := accounts.NewDirectoryStorage(args[0])
		if err != nil {
			cmd.PrintErrf("Unable to create directory storage: %s\n", err)
			os.Exit(1)
		}

		mux, err := streaming.NewMuxFromCredentialsDir(ds)
		if err != nil {
			cmd.PrintErrf("Unable to create mux: %s\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		events, errs := mux.StreamPublic(ctx, true)
		go func() {
			for {
				select {
				case serverError := <-errs:
					errJson, err := json.Marshal(serverError)
					if err != nil {
						cmd.PrintErrf("Unable to marshal error: %s\n", err)
						os.Exit(1)
					}
					cmd.Println(string(errJson))

				case event := <-events:
					eventJson, err := json.Marshal(event)
					if err != nil {
						cmd.PrintErrf("Unable to marshal event: %s\n", err)
						os.Exit(1)
					}
					asString := string(eventJson)
					// TODO: figure out what these are (keep alives?)
					if asString != "{}" {
						eventJson = append(eventJson, []byte("\n")...)
						cmd.Print(string(eventJson))
						_, err = gzWriter.Write(eventJson)
						if err != nil {
							cmd.PrintErrf("Unable to write to file: %s\n", err)
							os.Exit(1)
						}
					}
				}
			}
		}()

		waitForInterrupt()
	},
}
