package repository_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/internal/repository"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

const testDBAlias = "document_repo_test"

var (
	setupOnce sync.Once
	testOrm   orm.Ormer
)

func setupDocumentTestDB(t *testing.T) repository.DocumentRepo {
	t.Helper()
	setupOnce.Do(func() {
		config.Global = &config.Config{
			Database: config.DatabaseSection{Prefix: ""},
		}
		if err := orm.RegisterDataBase(testDBAlias, "sqlite3", ":memory:"); err != nil {
			panic(err)
		}
		orm.RegisterModelWithPrefix("", new(model.Document))
		if err := orm.RunSyncdb(testDBAlias, false, true); err != nil {
			panic(err)
		}
		testOrm = orm.NewOrmUsingDB(testDBAlias)
	})
	if _, err := testOrm.Raw("DELETE FROM documents").Exec(); err != nil {
		t.Fatalf("clear documents: %v", err)
	}
	return repository.NewDocumentRepo(testOrm)
}

func insertTestDocument(t *testing.T, o orm.Ormer, doc *model.Document) {
	t.Helper()
	id, err := o.Insert(doc)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}
	doc.DocumentId = int(id)
}

func TestDocumentRepo_Find(t *testing.T) {
	repo := setupDocumentTestDB(t)
	ctx := context.Background()

	doc := &model.Document{
		DocumentName: "hello",
		BookId:       1,
		Identify:     "doc-find",
		Markdown:     "body",
		MemberId:     1,
		Version:      100,
	}
	insertTestDocument(t, testOrm, doc)

	found, err := repo.Find(ctx, doc.DocumentId)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.DocumentName != "hello" || found.Version != 100 {
		t.Fatalf("unexpected document: %+v", found)
	}

	if _, err := repo.Find(ctx, 0); err != model.ErrInvalidParameter {
		t.Fatalf("expected ErrInvalidParameter for id=0, got %v", err)
	}
}

func TestDocumentRepo_UpdateMarkdownWithVersion_Success(t *testing.T) {
	repo := setupDocumentTestDB(t)
	ctx := context.Background()

	doc := &model.Document{
		DocumentName: "versioned",
		BookId:       1,
		Identify:     "doc-version",
		Markdown:     "old",
		MemberId:     1,
		ModifyAt:     1,
		Version:      200,
	}
	insertTestDocument(t, testOrm, doc)

	newVersion := time.Now().Unix()
	aff, err := repo.UpdateMarkdownWithVersion(ctx, doc.DocumentId, 200, "new content", 2, newVersion)
	if err != nil {
		t.Fatalf("UpdateMarkdownWithVersion: %v", err)
	}
	if aff != 1 {
		t.Fatalf("expected 1 row affected, got %d", aff)
	}

	updated, err := repo.Find(ctx, doc.DocumentId)
	if err != nil {
		t.Fatalf("Find after update: %v", err)
	}
	if updated.Markdown != "new content" || updated.Version != newVersion || updated.ModifyAt != 2 {
		t.Fatalf("unexpected updated document: %+v", updated)
	}
}

func TestDocumentRepo_UpdateMarkdownWithVersion_Conflict(t *testing.T) {
	repo := setupDocumentTestDB(t)
	ctx := context.Background()

	doc := &model.Document{
		DocumentName: "conflict",
		BookId:       1,
		Identify:     "doc-conflict",
		Markdown:     "keep",
		MemberId:     1,
		Version:      300,
	}
	insertTestDocument(t, testOrm, doc)

	aff, err := repo.UpdateMarkdownWithVersion(ctx, doc.DocumentId, 299, "stale", 1, 301)
	if err != nil {
		t.Fatalf("UpdateMarkdownWithVersion: %v", err)
	}
	if aff != 0 {
		t.Fatalf("expected 0 rows affected on version conflict, got %d", aff)
	}

	unchanged, err := repo.Find(ctx, doc.DocumentId)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if unchanged.Markdown != "keep" || unchanged.Version != 300 {
		t.Fatalf("document should be unchanged: %+v", unchanged)
	}
}

func TestDocumentRepo_FindByIdentify(t *testing.T) {
	repo := setupDocumentTestDB(t)
	ctx := context.Background()

	doc := &model.Document{
		DocumentName: "by-identify",
		BookId:       42,
		Identify:     "my-doc",
		Markdown:     "x",
		MemberId:     1,
		Version:      1,
	}
	insertTestDocument(t, testOrm, doc)

	found, err := repo.FindByIdentify(ctx, "my-doc", 42)
	if err != nil {
		t.Fatalf("FindByIdentify: %v", err)
	}
	if found.DocumentId != doc.DocumentId {
		t.Fatalf("expected document_id %d, got %d", doc.DocumentId, found.DocumentId)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
