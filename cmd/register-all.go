package cmd

import (
	"context"
	"github.com/abreka/proboscideans/accounts"
	"github.com/mattn/go-mastodon"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"time"
)

func initRegisterAllCmd() {
	initRegisterFlags(registerAllCmd)
}

var registerAllCmd = &cobra.Command{
	Use:   "register-all credentials-dir",
	Short: "register all instances by crawling peers breadth-first",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ds, err := accounts.NewDirectoryStorage(args[0])
		if err != nil {
			cmd.PrintErrf("Unable to create directory storage: %s\n", err)
			os.Exit(1)
		}

		existing, err := ds.GetAll()
		if err != nil {
			cmd.PrintErrf("Unable to get existing accounts: %s\n", err)
			os.Exit(1)
		} else if len(existing) == 0 {
			cmd.PrintErrln("No existing accounts found.")
			cmd.PrintErrln("Please register at least one account first via `probo register` command.")
			os.Exit(1)
		}

		visited := make(map[string]bool)
		frontier := make(map[string]bool)
		errored := make(map[string]bool)

		for {
			// Get the peers for every account we have that has not already been visited.
			for server, app := range existing {
				cmd.Printf("Getting peers for %s\n", server)
				if visited[server] {
					continue
				}

				func() {
					// TODO: error handling
					ctx, timeout := context.WithTimeout(context.Background(), 60*time.Second)
					defer timeout()

					client := mastodon.NewClient(&mastodon.Config{
						Server:       server,
						ClientID:     app.ClientID,
						ClientSecret: app.ClientSecret,
					})

					peers, err := client.GetInstancePeers(ctx)
					if err != nil {
						errored[server] = true
						cmd.PrintErrf("Unable to get peers for %s: %s\n", server, err)
						return
					}

					for _, peer := range peers {
						// Add https prefix if missing
						if !strings.HasPrefix(peer, "https://") {
							peer = "https://" + peer
						}

						if existing[peer] == nil && !errored[peer] {
							frontier[peer] = true
						}
					}
				}()

				visited[server] = true
			}

			// If the frontier is empty, we're done.
			if len(frontier) == 0 {
				cmd.Printf("No more peers to visit. Done.\n")
				break
			}

			// Register all the peers we found.
			for server := range frontier {
				cmd.Printf("Registering %s\n", server)

				func() {
					ctx, timeout := context.WithTimeout(context.Background(), 60*time.Second)
					defer timeout()

					app, err := mastodon.RegisterApp(ctx, &mastodon.AppConfig{
						Server:     server,
						ClientName: clientName,
						Scopes:     requiredAppScopes,
						Website:    appWebsite,
					})

					if err != nil {
						cmd.PrintErrf("Unable to register app: %s\n", err)
						errored[server] = true
						return
					}

					_, err = ds.WriteApp(server, app)
					if err != nil {
						cmd.PrintErrf("Unable to write app: %s\n", err)
						errored[server] = true
					}
				}()
			}

			// Clear the frontier.
			frontier = make(map[string]bool)

			// Refresh
			existing, err = ds.GetAll()
			if err != nil {
				cmd.PrintErrf("Unable to get existing accounts: %s\n", err)
				os.Exit(1)
			}
		}
	},
}
