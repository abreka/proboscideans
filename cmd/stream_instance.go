package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"

	"github.com/abreka/proboscideans/accounts"

	"github.com/abreka/proboscideans/streaming"
	"github.com/mattn/go-mastodon"
	"github.com/spf13/cobra"
)

var streamInstanceCmd = &cobra.Command{
	Use:   "stream-instance directory-storage server-name",
	Short: "stream events from a single instance",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		dirPath := args[0]
		serverName := args[1]

		ds, err := accounts.NewDirectoryStorage(dirPath)
		if err != nil {
			cmd.PrintErrf("Unable to create directory storage: %s\n", err)
			os.Exit(1)
		}

		app, err := ds.GetByServerName(serverName)
		if err != nil {
			cmd.PrintErrf("Unable to get app: %s\n", err)
			os.Exit(1)
		}

		server, err := streaming.ServerURIFromAppAuthURI(app)
		if err != nil {
			cmd.PrintErrf("Unable to get server from app: %s\n", err)
			os.Exit(1)
		}

		client := mastodon.NewClient(&mastodon.Config{
			Server:       server,
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// NOTE: I don't think you need to attach app credentials to stream.
		// However, it definitely seems much more polite.
		events, err := client.StreamingPublic(ctx, true)
		if err != nil {
			cmd.PrintErrf("Unable to stream: %s\n", err)
			os.Exit(1)
		}

		go func() {
			for event := range events {
				// Marshal as json and print it
				jsonEvent, err := json.Marshal(event)
				if err != nil {
					cmd.PrintErrf("Unable to marshal event: %s\n", err)
					os.Exit(1)
				}
				cmd.Println(string(jsonEvent))
			}
		}()

		waitForInterrupt()
	},
}

func waitForInterrupt() {
	killSignal := make(chan os.Signal, 1)
	signal.Notify(killSignal, os.Interrupt)
	<-killSignal
}
