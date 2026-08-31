package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/internal/repository"
)

func insertTestMemberAndToken(t *testing.T, account, email, hash string, expiresAt time.Time) (*model.Member, *model.MemberApiToken) {
	t.Helper()
	m := &model.Member{Account: account, Email: email, Password: "secret", Role: 2}
	id, err := testOrm.Insert(m)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	m.MemberId = int(id)

	tok := &model.MemberApiToken{
		MemberId:  m.MemberId,
		TokenHash: hash,
		Name:      "t",
		Scopes:    "read",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	tid, err := testOrm.Insert(tok)
	if err != nil {
		t.Fatalf("insert token: %v", err)
	}
	tok.TokenId = int(tid)
	return m, tok
}

func TestMemberRepo_ResolveAPITokenWithoutAside(t *testing.T) {
	_ = setupDocumentTestDB(t)
	repo := repository.NewMemberRepo(testOrm)
	ctx := context.Background()
	_, tok := insertTestMemberAndToken(t, "tok-plain", "tok-plain@example.com", "hash-plain", time.Time{})

	ident, err := repo.ResolveAPIToken(ctx, tok.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if ident.TokenID != tok.TokenId || ident.Member.Account != "tok-plain" {
		t.Fatalf("ident=%+v member=%+v", ident, ident.Member)
	}
	if ident.Member.Password != "" {
		t.Fatal("密码不应进入身份结果")
	}
}

func TestMemberRepo_ResolveAPITokenUsesAsideUntilInvalidate(t *testing.T) {
	_ = setupDocumentTestDB(t)
	setupAside(t)
	repo := repository.NewMemberRepo(testOrm)
	ctx := context.Background()
	m, tok := insertTestMemberAndToken(t, "tok-cached", "tok-cached@example.com", "hash-cached", time.Time{})

	first, err := repo.ResolveAPIToken(ctx, tok.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if first.Member.Account != "tok-cached" {
		t.Fatalf("account=%q", first.Member.Account)
	}

	if _, err := testOrm.Raw("UPDATE members SET account = ? WHERE member_id = ?", "mutated", m.MemberId).Exec(); err != nil {
		t.Fatal(err)
	}

	second, err := repo.ResolveAPIToken(ctx, tok.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if second.Member.Account != "tok-cached" {
		t.Fatalf("应命中缓存, got %q", second.Member.Account)
	}

	repository.InvalidateAPIToken(ctx, tok.TokenHash)
	third, err := repo.ResolveAPIToken(ctx, tok.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if third.Member.Account != "mutated" {
		t.Fatalf("失效后应回源, got %q", third.Member.Account)
	}
}

func TestMemberRepo_ResolveAPITokenNegativeCache(t *testing.T) {
	_ = setupDocumentTestDB(t)
	setupAside(t)
	repo := repository.NewMemberRepo(testOrm)
	ctx := context.Background()
	const missing = "hash-missing"

	if _, err := repo.ResolveAPIToken(ctx, missing); !errors.Is(err, model.ErrDataNotExist) {
		t.Fatalf("期望不存在, got %v", err)
	}

	_, tok := insertTestMemberAndToken(t, "tok-neg", "tok-neg@example.com", missing, time.Time{})
	if _, err := repo.ResolveAPIToken(ctx, missing); !errors.Is(err, model.ErrDataNotExist) {
		t.Fatalf("负缓存应仍未命中, got %v", err)
	}

	repository.InvalidateAPIToken(ctx, missing)
	got, err := repo.ResolveAPIToken(ctx, tok.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if got.TokenID != tok.TokenId {
		t.Fatalf("token_id=%d want %d", got.TokenID, tok.TokenId)
	}
}

func TestMemberRepo_ResolveAPITokenExpiredAndRevoked(t *testing.T) {
	_ = setupDocumentTestDB(t)
	setupAside(t)
	repo := repository.NewMemberRepo(testOrm)
	ctx := context.Background()

	_, expired := insertTestMemberAndToken(t, "tok-exp", "tok-exp@example.com", "hash-exp", time.Now().Add(-time.Hour))
	if _, err := repo.ResolveAPIToken(ctx, expired.TokenHash); !errors.Is(err, model.ErrDataNotExist) {
		t.Fatalf("过期 token 应拒绝, got %v", err)
	}

	_, live := insertTestMemberAndToken(t, "tok-rev", "tok-rev@example.com", "hash-rev", time.Time{})
	if _, err := repo.ResolveAPIToken(ctx, live.TokenHash); err != nil {
		t.Fatal(err)
	}
	if _, err := testOrm.Raw("UPDATE member_api_tokens SET revoked_at = ? WHERE token_id = ?", time.Now(), live.TokenId).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ResolveAPIToken(ctx, live.TokenHash); err != nil {
		t.Fatalf("吊销前缓存仍应命中, got %v", err)
	}
	repository.InvalidateAPIToken(ctx, live.TokenHash)
	if _, err := repo.ResolveAPIToken(ctx, live.TokenHash); !errors.Is(err, model.ErrDataNotExist) {
		t.Fatalf("吊销失效后应拒绝, got %v", err)
	}
}
