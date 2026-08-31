package cache

import (
	"context"
	"testing"
	"time"

	beegocache "github.com/beego/beego/v2/client/cache"
)

func TestAdapterRoundTrip(t *testing.T) {
	bm, err := beegocache.NewCache("memory", `{"interval":60}`)
	if err != nil {
		t.Fatal(err)
	}
	a := newAdapter(bm)
	ctx := context.Background()
	type payload struct {
		ID int `msgpack:"id"`
	}
	if err := a.Set(ctx, "k", payload{ID: 42}, time.Minute); err != nil {
		t.Fatal(err)
	}
	var got payload
	if err := a.Get(ctx, "k", &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 {
		t.Fatalf("got=%+v", got)
	}
	ok, err := a.IsExist(ctx, "k")
	if err != nil || !ok {
		t.Fatalf("exist=%v err=%v", ok, err)
	}
	if err := a.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if err := a.Get(ctx, "k", &got); err == nil {
		t.Fatal("删除后仍命中")
	}
}

func TestNullCacheNoop(t *testing.T) {
	n := &NullCache{}
	ctx := context.Background()
	if _, err := n.Get(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if err := n.Put(ctx, "k", "v", time.Second); err != nil {
		t.Fatal(err)
	}
	ok, err := n.IsExist(ctx, "k")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
