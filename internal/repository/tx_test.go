package repository_test

import (
	"context"
	"testing"

	"git.itopcms.com/astrueus/doc/internal/repository"
)

func TestWithOrm_RoundTrip(t *testing.T) {
	_ = setupDocumentTestDB(t)
	ctx := repository.WithOrm(context.Background(), testOrm)
	got := repository.OrmFromContext(ctx)
	if got != testOrm {
		t.Fatal("OrmFromContext did not return the injected Ormer")
	}
}

func TestUnitOfWork_RunPassesSameOrm(t *testing.T) {
	_ = setupDocumentTestDB(t)
	uow := repository.NewUnitOfWork()
	ctx := repository.WithOrm(context.Background(), testOrm)

	called := false
	err := uow.Run(ctx, func(inner context.Context) error {
		called = true
		if repository.OrmFromContext(inner) != testOrm {
			t.Fatal("UnitOfWork should keep the same Ormer in ctx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Fatal("fn was not called")
	}
}
