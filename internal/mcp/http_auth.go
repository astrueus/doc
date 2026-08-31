package mcp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/internal/repository"
)

var errUnauthorized = errors.New("unauthorized")

type authIdentity struct {
	Member    *model.Member
	TokenID   int
	TokenHash string
}

type tokenIDCtxKey struct{}

func withTokenID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, tokenIDCtxKey{}, id)
}

func tokenIDFromCtx(ctx context.Context) int {
	id, _ := ctx.Value(tokenIDCtxKey{}).(int)
	return id
}

// InvalidateAPITokenCache 吊销后删除 Token 缓存；等价于 repository.InvalidateAPIToken。
func InvalidateAPITokenCache(ctx context.Context, tokenHash string) {
	repository.InvalidateAPIToken(ctx, tokenHash)
}

func verifyBearer(r *http.Request, cfg *config.MCPSection) (*authIdentity, error) {
	if cfg == nil {
		cfg = &config.MustGlobal().MCP
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		if cfg.TokenRequired {
			return nil, errUnauthorized
		}
		return identityFromStdioMember(cfg)
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, errUnauthorized
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if !strings.HasPrefix(token, "doc_") {
		return nil, errUnauthorized
	}
	raw := strings.TrimPrefix(token, "doc_")
	if raw == "" {
		return nil, errUnauthorized
	}

	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])

	ctx := repository.ContextWithClientIP(r.Context(), clientIP(r))
	ident, err := memberRepo().ResolveAPIToken(ctx, hash)
	if err != nil || ident == nil || ident.Member == nil || ident.Member.MemberId <= 0 {
		return nil, errUnauthorized
	}
	return &authIdentity{Member: ident.Member, TokenID: ident.TokenID, TokenHash: hash}, nil
}

func identityFromStdioMember(cfg *config.MCPSection) (*authIdentity, error) {
	account := strings.TrimSpace(cfg.StdioMember)
	if account == "" {
		return nil, errUnauthorized
	}
	member, err := memberRepo().FindByAccount(context.Background(), account)
	if err != nil {
		return nil, errUnauthorized
	}
	return &authIdentity{Member: member, TokenID: 0, TokenHash: ""}, nil
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func classifyToolKindFromBody(body []byte) string {
	if len(body) == 0 {
		return "read"
	}
	var msg struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "read"
	}
	if msg.Method != "tools/call" {
		return "read"
	}
	switch msg.Params.Name {
	case "delete_document":
		return "delete"
	case "create_document", "update_document_content", "append_document_content",
		"update_document_meta", "release_document", "create_book", "update_book":
		return "write"
	default:
		return "read"
	}
}

func peekRequestBody(r *http.Request, max int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, max))
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(data))
	return data, err
}
