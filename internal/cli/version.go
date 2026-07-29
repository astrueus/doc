package cli

import (
	"fmt"

	"git.itopcms.com/jackliu/doc/internal/config"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Doc version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Doc current version =>", config.VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
