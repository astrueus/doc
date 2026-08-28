package mcp

import (
	"context"

	"git.itopcms.com/astrueus/doc/internal/model"
)

type searchProvider interface {
	Search(ctx context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error)
}

type sqlLikeProvider struct{}

func newSearchProvider() searchProvider {
	return &sqlLikeProvider{}
}

func (p *sqlLikeProvider) Search(ctx context.Context, q string, bookIDs []int, limit int) ([]*model.Document, error) {
	return documentRepo().SearchLike(ctx, q, bookIDs, limit)
}
