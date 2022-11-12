package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/abreka/proboscideans/accounts"

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
	registerInstance.Flags().StringVar(&server, "server", "https://mastodon.social", "The server to register the app with")
	registerInstance.Flags().StringVar(&clientName, "client-name", "proboscideans", "The name of the app")
	registerInstance.Flags().StringVar(&requiredAppScopes, "scopes", "read", "The scopes required by the app")
	registerInstance.Flags().StringVar(&appWebsite, "website", "https://twitter.com/generativist", "The website of the app")
}

// registerInstance represents the register command
var registerInstance = &cobra.Command{
	Use:   "register-instance [output-dir]",
	Short: "register a new app with an instance",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var ds *accounts.DirectoryStore

		if len(args) == 1 {
			ds, err = accounts.NewDirectoryStorage(args[0])
			if err != nil {
				cmd.PrintErrf("Unable to create directory storage: %s", err)
				os.Exit(1)
			}
		}

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
			outputPath, err := ds.WriteApp(server, app)
			if err != nil {
				cmd.PrintErrf("Unable to write app: %s", err)
				os.Exit(1)
			}

			cmd.Printf("Wrote app to %s", outputPath)
		}
	},
}
