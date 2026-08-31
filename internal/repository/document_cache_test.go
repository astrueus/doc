package repository_test

import (
	"context"
	"testing"

	"git.itopcms.com/astrueus/doc/internal/cache"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/internal/repository"
)

func setupAside(t *testing.T) {
	t.Helper()
	rt, err := cache.Open(context.Background(), cache.Settings{
		Mode:          cache.ModeLocal,
		L1MaxCost:     1 << 20,
		L1NumCounters: 1_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	cache.SetKernel(rt)
	repository.RegisterCacheInvalidation()
	t.Cleanup(func() { cache.SetKernel(nil) })
}

func TestDocumentRepo_FindUsesAsideUntilInvalidate(t *testing.T) {
	repo := setupDocumentTestDB(t)
	setupAside(t)
	ctx := context.Background()

	doc := &model.Document{
		DocumentName: "cached",
		BookId:       3,
		Identify:     "c1",
		Markdown:     "m",
		MemberId:     1,
		Version:      1,
	}
	insertTestDocument(t, testOrm, doc)

	first, err := repo.Find(ctx, doc.DocumentId)
	if err != nil {
		t.Fatal(err)
	}
	if first.DocumentName != "cached" {
		t.Fatalf("name=%q", first.DocumentName)
	}

	if _, err := testOrm.Raw("UPDATE documents SET document_name = ? WHERE document_id = ?", "mutated", doc.DocumentId).Exec(); err != nil {
		t.Fatal(err)
	}

	second, err := repo.Find(ctx, doc.DocumentId)
	if err != nil {
		t.Fatal(err)
	}
	if second.DocumentName != "cached" {
		t.Fatalf("应命中缓存, got %q", second.DocumentName)
	}

	repository.InvalidateDocument(first)
	third, err := repo.Find(ctx, doc.DocumentId)
	if err != nil {
		t.Fatal(err)
	}
	if third.DocumentName != "mutated" {
		t.Fatalf("失效后应回源, got %q", third.DocumentName)
	}
}

func TestDocumentRepo_FindByIdentifyCached(t *testing.T) {
	repo := setupDocumentTestDB(t)
	setupAside(t)
	ctx := context.Background()

	doc := &model.Document{
		DocumentName: "ident",
		BookId:       5,
		Identify:     "by-ident",
		Markdown:     "m",
		MemberId:     1,
		Version:      1,
	}
	insertTestDocument(t, testOrm, doc)

	first, err := repo.FindByIdentify(ctx, "by-ident", 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testOrm.Raw("UPDATE documents SET document_name = ? WHERE document_id = ?", "x", doc.DocumentId).Exec(); err != nil {
		t.Fatal(err)
	}
	second, err := repo.FindByIdentify(ctx, "by-ident", 5)
	if err != nil {
		t.Fatal(err)
	}
	if second.DocumentName != first.DocumentName {
		t.Fatalf("identify 应命中缓存, got %q", second.DocumentName)
	}
}

func TestBlogRepo_FindUsesAsideUntilInvalidate(t *testing.T) {
	_ = setupDocumentTestDB(t)
	setupAside(t)
	ctx := context.Background()

	blog := &model.Blog{
		BlogTitle:    "hello",
		BlogIdentify: "b1",
		MemberId:     0,
		BlogType:     0,
		BlogStatus:   "publish",
		BlogContent:  "c",
		BlogRelease:  "r",
		Version:      1,
	}
	id, err := testOrm.Insert(blog)
	if err != nil {
		t.Fatalf("insert blog: %v", err)
	}
	blog.BlogId = int(id)

	repo := repository.NewBlogRepo(testOrm)
	first, err := repo.Find(ctx, blog.BlogId)
	if err != nil {
		t.Fatal(err)
	}
	if first.BlogTitle != "hello" {
		t.Fatalf("title=%q", first.BlogTitle)
	}

	if _, err := testOrm.Raw("UPDATE blogs SET blog_title = ? WHERE blog_id = ?", "mutated", blog.BlogId).Exec(); err != nil {
		t.Fatal(err)
	}

	second, err := repo.Find(ctx, blog.BlogId)
	if err != nil {
		t.Fatal(err)
	}
	if second.BlogTitle != "hello" {
		t.Fatalf("应命中缓存, got %q", second.BlogTitle)
	}

	repository.InvalidateBlog(first)
	third, err := repo.Find(ctx, blog.BlogId)
	if err != nil {
		t.Fatal(err)
	}
	if third.BlogTitle != "mutated" {
		t.Fatalf("失效后应回源, got %q", third.BlogTitle)
	}
}
