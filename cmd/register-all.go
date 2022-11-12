package cmd

import "github.com/spf13/cobra"

var registerAllCmd = &cobra.Command{
	Use:   "register-all credentials-dir",
	Short: "register all instances by crawling peers",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

	},
}
