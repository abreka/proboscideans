package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/abreka/proboscideans/accounts"
	"github.com/spf13/cobra"
)

func initRegisterAllCmd() {
	initRegisterFlags(registerAllCmd)
}

var registerAllCmd = &cobra.Command{
	Use:   "register-all credentials-dir snapshot_path",
	Short: "register all instances by crawling peers breadth-first",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: cleanup refactor
		// This code is ugly as shit
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
		concurrency := 128

		snapshotPath := args[1]
		if snapshotPath == "now.jsonl" {
			snapshotPath = fmt.Sprintf("%d.jsonl", time.Now().Unix())
		}

		snapshotFp, err := os.OpenFile(snapshotPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			cmd.PrintErrf("Unable to open snapshot file: %s\n", err)
			os.Exit(1)
		}

		for {
			skip := make(map[string]bool)
			for server := range errored {
				skip[server] = true
			}
			for server := range visited {
				skip[server] = true
			}

			peersCh := accounts.GetAllPeers(
				context.Background(), existing, concurrency, visited, 10*time.Second,
			)

			for peer := range peersCh {
				visited[peer.Server] = true

				if peer.Err != nil {
					cmd.PrintErrf("Error getting peers for %s: %s\n", peer.Server, peer.Err)
					errored[peer.Server] = true
					continue
				}

				cmd.Printf("Got %d peers for %s\n", len(peer.Peers), peer.Server)

				// Write to snapshot file
				b, err := json.Marshal(peer)
				if err != nil {
					cmd.PrintErrf("Error marshalling peer: %s\n", err)
					os.Exit(1)
				}
				b = append(b, []byte("\n")...)
				_, err = snapshotFp.Write(b)
				if err != nil {
					cmd.PrintErrf("Error writing to snapshot file: %s\n", err)
					os.Exit(1)
				}

				for _, peer := range peer.Peers {
					// Add https prefix if missing
					if !strings.HasPrefix(peer, "https://") {
						peer = "https://" + peer
					}

					if existing[peer] == nil && !errored[peer] {
						frontier[peer] = true
					}
				}
			}

			// If the frontier is empty, we're done.
			if len(frontier) == 0 {
				cmd.Printf("No more peers to visit. Done.\n")
				break
			}

			cmd.Printf("Visiting %d new peers\n", len(frontier))

			registrations := accounts.RegisterAll(
				context.Background(),
				frontier,
				concurrency,
				10*time.Second,
				clientName,
				requiredAppScopes,
				appWebsite,
			)

			nAttemped := 0
			for registration := range registrations {
				nAttemped++
				if registration.Err != nil {
					cmd.PrintErrf("Error registering %s: %s\n", registration.Server, registration.Err)
					errored[registration.Server] = true
					continue
				}

				_, err = ds.WriteApp(registration.Server, registration.App)
				if err != nil {
					cmd.PrintErrf("Unable to write app: %s\n", err)
					errored[registration.Server] = true
				}

				cmd.Printf("Registered %s\n", registration.Server)
			}

			cmd.Printf("Attempted to register %d peers\n", nAttemped)

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
