package mcp

import (
	"context"
	"fmt"
	"strings"

	"git.itopcms.com/astrueus/doc/internal/config"
	"github.com/beego/beego/v2/core/logs"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio starts the MCP server on stdin/stdout using the configured stdio member identity.
func RunStdio(ctx context.Context, cfg *config.MCPSection) error {
	if cfg == nil {
		return fmt.Errorf("mcp config is nil")
	}
	account := strings.TrimSpace(cfg.StdioMember)
	if account == "" {
		return fmt.Errorf("mcp_stdio_member is empty")
	}
	member, err := memberRepo().FindByAccount(ctx, account)
	if err != nil {
		return fmt.Errorf("stdio member %q not found: %w", account, err)
	}

	ctx = withMember(ctx, member)
	// Belt-and-suspenders: cli already called app.SuppressConsoleLogger before bootstrap.
	_ = logs.GetBeeLogger().DelLogger("console")
	srv := newServer()
	return srv.Run(ctx, &sdkmcp.StdioTransport{})
}
