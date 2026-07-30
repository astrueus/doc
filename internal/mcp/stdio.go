package mcp

import (
	"context"
	"fmt"
	"strings"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/errs"
	"git.itopcms.com/jackliu/doc/internal/model"
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
	member, err := model.NewMember().FindByAccount(account)
	if err != nil {
		return fmt.Errorf("stdio member %q not found: %w", account, err)
	}

	ctx = withMember(ctx, member)
	quietLogsForStdio()
	srv := newServer()
	return srv.Run(ctx, &sdkmcp.StdioTransport{})
}

func quietLogsForStdio() {
	_ = logs.GetBeeLogger().DelLogger("console")
}

// RunHTTP is implemented in T5 (Streamable HTTP + Bearer).
func RunHTTP(ctx context.Context, cfg *config.MCPSection) error {
	return errs.New(errs.CodeInternal, "MCP HTTP mode is planned for Round 3 T5")
}
