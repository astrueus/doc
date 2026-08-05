package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"git.itopcms.com/astrueus/doc/internal/config"
	"github.com/beego/beego/v2/core/logs"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	httpServerOnce sync.Once
	sharedHTTPSrv  *sdkmcp.Server
	streamHandler  http.Handler
)

func sharedHTTPServer() *sdkmcp.Server {
	httpServerOnce.Do(func() {
		configureRateLimiterFromConfig()
		sharedHTTPSrv = newServer()
		// DisableLocalhostProtection: Nginx/Traefik 反代到 127.0.0.1 时，
		// LocalAddr 为 loopback 但 Host 为公网域名，SDK 默认会 403
		// "Forbidden: invalid Host header"。HTTP MCP 已有 Bearer 鉴权，可安全关闭。
		streamHandler = sdkmcp.NewStreamableHTTPHandler(
			func(*http.Request) *sdkmcp.Server { return sharedHTTPSrv },
			&sdkmcp.StreamableHTTPOptions{
				Stateless:                    true,
				JSONResponse:                 true,
				DisableLocalhostProtection:   true,
			},
		)
	})
	return sharedHTTPSrv
}

// NewHTTPHandler returns an http.Handler with Bearer auth + rate limiting
// wrapping the official Streamable HTTP transport.
func NewHTTPHandler() http.Handler {
	_ = sharedHTTPServer()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := &config.MustGlobal().MCP
		id, err := verifyBearer(r, cfg)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, _ := peekRequestBody(r, 1<<20)
		kind := classifyToolKindFromBody(body)
		if !AllowByToken(id.TokenID, kind) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}

		ctx := withMember(r.Context(), id.Member)
		ctx = withTokenID(ctx, id.TokenID)
		streamHandler.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RunHTTP starts a dedicated Streamable HTTP MCP server on cfg.Listen.
func RunHTTP(ctx context.Context, cfg *config.MCPSection) error {
	if cfg == nil {
		return fmt.Errorf("mcp config is nil")
	}
	addr := strings.TrimSpace(cfg.Listen)
	if addr == "" {
		addr = "127.0.0.1:8280"
	}
	warnInsecureListen(addr)

	handler := NewHTTPHandler()
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	logs.Info("MCP Streamable HTTP listening on %s (token_required=%v rate_limit=%d)", addr, cfg.TokenRequired, cfg.RateLimit)
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func warnInsecureListen(addr string) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	if host == "0.0.0.0" || host == "::" || host == "" {
		logs.Warning("MCP listen is %s without detected TLS: Bearer tokens must not be sent over plain HTTP; put HTTPS (Nginx/Traefik) in front", addr)
	}
}
