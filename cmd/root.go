package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "probo",
	Short: "tools for consuming mastodons",
	Long:  `probo is a collection of tools for consuming mastodon instances`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.UsageString())
	},
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Add subcommands
	rootCmd.AddCommand(registerCmd)

	// Add flags
	initRegisterCmd()
}

// Execute runs the CLI app
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
