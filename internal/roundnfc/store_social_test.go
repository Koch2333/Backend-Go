package roundnfc

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSocialLinksRoundTrip(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "roundnfc.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	items, err := store.ListSocialLinks(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("expected 5 default social links, got %d", len(items))
	}

	for i := range items {
		if items[i].Key == "bilibili" {
			items[i].Value = "测试账号"
			items[i].URL = "https://space.bilibili.com/123"
			items[i].Enabled = true
		}
	}
	if err := store.ReplaceSocialLinks(ctx, items); err != nil {
		t.Fatal(err)
	}

	visible, err := store.ListSocialLinks(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].Key != "bilibili" || visible[0].URL == "" {
		t.Fatalf("unexpected visible social links: %#v", visible)
	}
}
