package cli

import (
	"context"
	"fmt"

	"git.itopcms.com/astrueus/doc/internal/app"
	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/mcp"
	"github.com/spf13/cobra"
)

var httpMode bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !httpMode {
			// Must run before bootstrap: RegisterLogger / DB init otherwise pollute stdout.
			app.SuppressConsoleLogger()
		}
		bootstrapFromFlags()
		ctx := context.Background()
		cfg := &config.MustGlobal().MCP
		if httpMode {
			if !cfg.Enable {
				fmt.Println("[warn] mcp_enable=false but --http was requested; starting HTTP MCP anyway")
			}
			return mcp.RunHTTP(ctx, cfg)
		}
		return mcp.RunStdio(ctx, cfg)
	},
}

func init() {
	mcpCmd.Flags().BoolVar(&httpMode, "http", false, "Serve MCP over Streamable HTTP (T5)")
	rootCmd.AddCommand(mcpCmd)
}
