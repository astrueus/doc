package repository

import (
	"context"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache"
	"git.itopcms.com/astrueus/doc/internal/model"
)

func metaCacheOptions(tags ...string) cache.Options {
	return cache.Options{
		TTL:       10 * time.Minute,
		SoftTTL:   8 * time.Minute,
		L1TTL:     20 * time.Second,
		CacheNull: true,
		NullTTL:   45 * time.Second,
		Tags:      tags,
	}
}

func cacheKeys() cache.KeyBuilder {
	if rt := cache.Kernel(); rt != nil {
		return rt.Keys
	}
	return cache.Keys()
}

func documentAside() *cache.Aside[model.Document] {
	rt := cache.Kernel()
	if rt == nil {
		return nil
	}
	return cache.NewAsideFrom[model.Document](rt)
}

func blogAside() *cache.Aside[model.Blog] {
	rt := cache.Kernel()
	if rt == nil {
		return nil
	}
	return cache.NewAsideFrom[model.Blog](rt)
}

// RegisterCacheInvalidation 把 model 写路径接到 Aside 失效。
func RegisterCacheInvalidation() {
	model.AfterDocumentMutated = InvalidateDocument
	model.AfterBlogMutated = InvalidateBlog
}

// InvalidateDocument 删除文档 id / identify 键，并按 book、document tag 失效。
func InvalidateDocument(doc *model.Document) {
	if doc == nil || doc.DocumentId <= 0 {
		return
	}
	a := documentAside()
	if a == nil {
		return
	}
	ctx := context.Background()
	k := cacheKeys()
	keys := []string{k.DocumentByID(doc.DocumentId)}
	if doc.Identify != "" && doc.BookId > 0 {
		keys = append(keys, k.DocumentByIdentify(doc.BookId, doc.Identify))
	}
	_ = a.Delete(ctx, keys...)
	tags := []string{k.TagDocument(doc.DocumentId)}
	if doc.BookId > 0 {
		tags = append(tags, k.TagBook(doc.BookId))
	}
	_ = a.InvalidateTag(ctx, tags...)
}

// InvalidateBlog 删除博客主键键。
func InvalidateBlog(blog *model.Blog) {
	if blog == nil || blog.BlogId <= 0 {
		return
	}
	a := blogAside()
	if a == nil {
		return
	}
	ctx := context.Background()
	k := cacheKeys()
	_ = a.Delete(ctx, k.BlogByID(blog.BlogId))
	_ = a.InvalidateTag(ctx, k.TagBlog(blog.BlogId))
}
