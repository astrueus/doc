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
	"time"

	"git.itopcms.com/jackliu/doc/internal/cache"
	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
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

func tokenCacheKey(hash string) string {
	return "mcp:tok:" + hash
}

// InvalidateAPITokenCache drops the Bearer auth cache entry for a token hash.
func InvalidateAPITokenCache(ctx context.Context, tokenHash string) {
	if cache.Default == nil || tokenHash == "" {
		return
	}
	_ = cache.Delete(ctx, tokenCacheKey(tokenHash))
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

	if cache.Default != nil {
		var cached model.Member
		if err := cache.Get(r.Context(), tokenCacheKey(hash), &cached); err == nil && cached.MemberId > 0 {
			// Cache stores member only; re-resolve token id lightly from DB if needed for rate limit.
			t, err := model.NewMemberApiToken().FindByHash(hash)
			if err == nil && !t.IsRevoked() && !t.IsExpired(time.Now()) {
				return &authIdentity{Member: &cached, TokenID: t.TokenId, TokenHash: hash}, nil
			}
			_ = cache.Delete(r.Context(), tokenCacheKey(hash))
		}
	}

	t, err := model.NewMemberApiToken().FindByHash(hash)
	if err != nil {
		return nil, errUnauthorized
	}
	if t.IsRevoked() || t.IsExpired(time.Now()) {
		return nil, errUnauthorized
	}

	member, err := model.NewMember().Find(t.MemberId)
	if err != nil || member.MemberId <= 0 {
		return nil, errUnauthorized
	}

	go updateLastUsed(t.TokenId, clientIP(r))

	if cache.Default != nil {
		_ = cache.Set(r.Context(), tokenCacheKey(hash), *member, 5*time.Minute)
	}

	return &authIdentity{Member: member, TokenID: t.TokenId, TokenHash: hash}, nil
}

func identityFromStdioMember(cfg *config.MCPSection) (*authIdentity, error) {
	account := strings.TrimSpace(cfg.StdioMember)
	if account == "" {
		return nil, errUnauthorized
	}
	member, err := model.NewMember().FindByAccount(account)
	if err != nil {
		return nil, errUnauthorized
	}
	return &authIdentity{Member: member, TokenID: 0, TokenHash: ""}, nil
}

func updateLastUsed(tokenID int, ip string) {
	defer func() {
		if rec := recover(); rec != nil {
			logs.Error("updateLastUsed panic: %v", rec)
		}
	}()
	o := orm.NewOrm()
	_, err := o.QueryTable(model.NewMemberApiToken().TableNameWithPrefix()).
		Filter("token_id", tokenID).
		Update(orm.Params{
			"last_used_at": time.Now(),
			"last_used_ip": ip,
		})
	if err != nil {
		logs.Warning("updateLastUsed token=%d: %v", tokenID, err)
	}
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
		"update_document_meta", "release_document":
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
