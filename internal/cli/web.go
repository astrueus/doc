package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// StartWebServer is injected by main to avoid an import cycle with commands/daemon.
var StartWebServer func(flagArgs []string) error

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the HTTP web server",
	RunE:  runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)
}

func runWeb(cmd *cobra.Command, args []string) error {
	if StartWebServer == nil {
		return errors.New("web server starter is not registered")
	}
	return StartWebServer(buildFlagArgs())
}
