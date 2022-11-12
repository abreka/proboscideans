package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/abreka/proboscideans/streaming"
	"github.com/spf13/cobra"
)

var streamDistributedCmd = &cobra.Command{
	Use:   "stream-distributed [credentials-dir]",
	Short: "stream events from multiple instances",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mux, err := streaming.NewMuxFromCredentialsDir(args[0])
		if err != nil {
			cmd.PrintErrf("Unable to create mux: %s", err)
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
						cmd.PrintErrf("Unable to marshal error: %s", err)
						os.Exit(1)
					}
					cmd.Println(string(errJson))
				case event := <-events:
					eventJson, err := json.Marshal(event)
					if err != nil {
						cmd.PrintErrf("Unable to marshal event: %s", err)
						os.Exit(1)
					}
					cmd.Println(string(eventJson))
				}
			}
		}()

		waitForInterrupt()
	},
}
