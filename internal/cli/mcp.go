package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (planned for Round 3)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("mcp command is planned for Round 3, not implemented yet")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
