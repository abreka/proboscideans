package cmd

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/mattn/go-mastodon"
	"github.com/spf13/cobra"
)

var whoisCmd = &cobra.Command{
	Use:   "whois",
	Short: "whois an instance",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mastodon.NewClient(&mastodon.Config{
			Server: args[0],
		})

		ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
		defer timeout()

		info, err := client.GetInstance(ctx)
		if err != nil {
			cmd.PrintErrf("Unable to get instance info: %s", err)
			os.Exit(1)
		}

		asJson, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			cmd.PrintErrf("Unable to marshal instance info: %s", err)
			os.Exit(1)
		}

		cmd.Println(string(asJson))
	},
}
