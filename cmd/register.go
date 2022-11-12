package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mattn/go-mastodon"
	"github.com/spf13/cobra"
)

var (
	server            string
	clientName        string
	requiredAppScopes string
	appWebsite        string
)

func initRegisterCmd() {
	registerCmd.Flags().StringVar(&server, "server", "https://mastodon.social", "The server to register the app with")
	registerCmd.Flags().StringVar(&clientName, "client-name", "proboscideans", "The name of the app")
	registerCmd.Flags().StringVar(&requiredAppScopes, "scopes", "read", "The scopes required by the app")
	registerCmd.Flags().StringVar(&appWebsite, "website", "https://twitter.com/generativist", "The website of the app")
}

// registerCmd represents the register command
var registerCmd = &cobra.Command{
	Use:   "register [output-path]",
	Short: "register a new app",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
			Server:     server,
			ClientName: clientName,
			Scopes:     requiredAppScopes,
			Website:    appWebsite,
		})

		if err != nil {
			cmd.PrintErrf("Unable to register app: %s", err)
			os.Exit(1)
		}

		asJson, err := json.MarshalIndent(app, "", "  ")
		if err != nil {
			cmd.PrintErrf("Unable to marshal app: %s", err)
			os.Exit(1)
		}

		if len(args) == 0 {
			cmd.Println(string(asJson))
		} else {
			err = os.WriteFile(args[0], asJson, 0644)
			if err != nil {
				cmd.PrintErrf("Unable to write app to file: %s", err)
				os.Exit(1)
			}
		}
	},
}
