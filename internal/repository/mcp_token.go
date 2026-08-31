package repository

import (
	"context"
	"errors"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

type clientIPCtxKey struct{}

// APITokenIdentity 是 MCP Bearer 校验结果（不含明文 token）。
type APITokenIdentity struct {
	Member  *model.Member
	TokenID int
}

// mcpTokenCacheValue 为 Token→成员 的 Aside 载荷；不含密码。
type mcpTokenCacheValue struct {
	Member    model.Member
	TokenID   int
	ExpiresAt time.Time
}

func tokenCacheOptions() cache.Options {
	return cache.Options{
		TTL:       5 * time.Minute,
		SoftTTL:   4 * time.Minute,
		L1TTL:     15 * time.Second,
		CacheNull: true,
		NullTTL:   30 * time.Second,
	}
}

func mcpTokenAside() *cache.Aside[mcpTokenCacheValue] {
	rt := cache.Kernel()
	if rt == nil {
		return nil
	}
	return cache.NewAsideFrom[mcpTokenCacheValue](rt)
}

// ContextWithClientIP 把客户端 IP 放入 ctx，供 Token 回源时异步更新 last_used。
func ContextWithClientIP(ctx context.Context, ip string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, clientIPCtxKey{}, ip)
}

func clientIPFromCtx(ctx context.Context) string {
	s, _ := ctx.Value(clientIPCtxKey{}).(string)
	return s
}

// InvalidateAPIToken 删除 MCP Token 缓存键（吊销后必须调用）。
func InvalidateAPIToken(ctx context.Context, tokenHash string) {
	if tokenHash == "" {
		return
	}
	a := mcpTokenAside()
	if a == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	_ = a.Delete(ctx, cacheKeys().MCPToken(tokenHash))
}

func (v mcpTokenCacheValue) expired(now time.Time) bool {
	return !v.ExpiresAt.IsZero() && v.ExpiresAt.Before(now)
}

func tokenIdentityFromCache(v mcpTokenCacheValue) *APITokenIdentity {
	m := v.Member
	m.Password = ""
	m.ResolveRoleName()
	return &APITokenIdentity{Member: &m, TokenID: v.TokenID}
}

func (r *memberRepo) ResolveAPIToken(ctx context.Context, tokenHash string) (*APITokenIdentity, error) {
	if tokenHash == "" {
		return nil, model.ErrDataNotExist
	}
	if a := mcpTokenAside(); a != nil {
		key := cacheKeys().MCPToken(tokenHash)
		v, err := a.GetOrLoad(ctx, key, tokenCacheOptions(), func(context.Context) (mcpTokenCacheValue, error) {
			ident, err := r.loadAPITokenIdentity(ctx, tokenHash)
			if err != nil {
				if errors.Is(err, model.ErrDataNotExist) {
					return mcpTokenCacheValue{}, cache.ErrNotFound
				}
				return mcpTokenCacheValue{}, err
			}
			r.touchLastUsedAsync(ident.TokenID, clientIPFromCtx(ctx))
			return mcpTokenCacheValue{
				Member:    *ident.Member,
				TokenID:   ident.TokenID,
				ExpiresAt: ident.expiresAt,
			}, nil
		})
		if errors.Is(err, cache.ErrNotFound) {
			return nil, model.ErrDataNotExist
		}
		if err != nil {
			return nil, err
		}
		if v.expired(time.Now()) {
			_ = a.Delete(ctx, key)
			return nil, model.ErrDataNotExist
		}
		return tokenIdentityFromCache(v), nil
	}
	ident, err := r.loadAPITokenIdentity(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	r.touchLastUsedAsync(ident.TokenID, clientIPFromCtx(ctx))
	return ident.APITokenIdentity, nil
}

// tokenIdentityLoad 为回源结果，带过期时间供写入缓存。
type tokenIdentityLoad struct {
	*APITokenIdentity
	expiresAt time.Time
}

func (r *memberRepo) loadAPITokenIdentity(ctx context.Context, hash string) (*tokenIdentityLoad, error) {
	tok, err := r.FindAPITokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, orm.ErrNoRows) {
			return nil, model.ErrDataNotExist
		}
		return nil, err
	}
	now := time.Now()
	if tok.IsRevoked() || tok.IsExpired(now) {
		return nil, model.ErrDataNotExist
	}
	member, err := r.Find(ctx, tok.MemberId)
	if err != nil || member == nil || member.MemberId <= 0 {
		return nil, model.ErrDataNotExist
	}
	member.Password = ""
	member.Lang = ""
	return &tokenIdentityLoad{
		APITokenIdentity: &APITokenIdentity{Member: member, TokenID: tok.TokenId},
		expiresAt:        tok.ExpiresAt,
	}, nil
}

func (r *memberRepo) touchLastUsedAsync(tokenID int, ip string) {
	if tokenID <= 0 {
		return
	}
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				logs.Error("TouchAPITokenLastUsed panic: %v", rec)
			}
		}()
		if err := r.TouchAPITokenLastUsed(context.Background(), tokenID, ip); err != nil {
			logs.Warning("TouchAPITokenLastUsed token=%d: %v", tokenID, err)
		}
	}()
}
