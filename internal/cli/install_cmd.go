package cli

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Initialize database and default data",
	Run: func(cmd *cobra.Command, args []string) {
		bootstrapFromFlags()
		Install()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
