package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strings"

	"github.com/mattn/go-mastodon"
	"github.com/spf13/cobra"
)

var streamCmd = &cobra.Command{
	Use:   "stream [credentials-path]",
	Short: "stream events from the server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		app, err := LoadAppFromJSON(args[0])
		if err != nil {
			cmd.PrintErrf("Unable to load app from JSON: %s", err)
			os.Exit(1)
		}

		// Find the "/oauth/" in the auth uri and use that as the server.
		// This is a KLUDGE to work around the fact that the app doesn't
		// store the server but I think they all have the same url structure
		// so it should be fine for now...until it isn.t
		i := strings.Index(app.AuthURI, "/oauth/")
		if i == -1 {
			cmd.PrintErrf("Unable to find /oauth/ in auth uri: %s", app.AuthURI)
			os.Exit(1)
		}
		server := app.AuthURI[:i]

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
			cmd.PrintErrf("Unable to stream: %s", err)
			os.Exit(1)
		}

		go func() {
			for event := range events {
				// Marshal as json and print it
				jsonEvent, err := json.Marshal(event)
				if err != nil {
					cmd.PrintErrf("Unable to marshal event: %s", err)
					os.Exit(1)
				}
				cmd.Println(string(jsonEvent))
			}
		}()

		waitForInterrupt()
	},
}

// LoadAppFromJSON loads an app from a JSON file
func LoadAppFromJSON(filePath string) (*mastodon.Application, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	var app mastodon.Application
	if err := json.NewDecoder(fp).Decode(&app); err != nil {
		return nil, err
	}

	return &app, nil
}

func waitForInterrupt() {
	killSignal := make(chan os.Signal, 1)
	signal.Notify(killSignal, os.Interrupt)
	<-killSignal
}
