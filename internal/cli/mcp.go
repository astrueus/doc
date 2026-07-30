package cli

import (
	"context"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/mcp"
	"github.com/spf13/cobra"
)

var httpMode bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		bootstrapFromFlags()
		ctx := context.Background()
		if httpMode {
			return mcp.RunHTTP(ctx, &config.MustGlobal().MCP)
		}
		return mcp.RunStdio(ctx, &config.MustGlobal().MCP)
	},
}

func init() {
	mcpCmd.Flags().BoolVar(&httpMode, "http", false, "Serve MCP over Streamable HTTP (T5)")
	rootCmd.AddCommand(mcpCmd)
}
