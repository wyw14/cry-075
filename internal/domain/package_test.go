package domain

import "testing"

func TestPackageReorderKeepsPinnedItemsFirst(t *testing.T) {
	pack, _ := NewAssetPackage("夏末组合", "production", false)
	a, b, c := ID("a"), ID("b"), ID("c")
	_ = pack.Add(a, "", false)
	_ = pack.Add(b, "", true)
	_ = pack.Add(c, "", false)
	if err := pack.Reorder([]ID{c, a, b}); err != nil {
		t.Fatal(err)
	}
	if pack.Items[0].AssetID != b || !pack.Items[0].Pinned {
		t.Fatalf("pinned asset should remain first: %#v", pack.Items)
	}
	seen := map[ID]bool{}
	for index, item := range pack.Items {
		if seen[item.AssetID] {
			t.Fatal("duplicate asset")
		}
		seen[item.AssetID] = true
		if item.Position != index+1 {
			t.Fatal("positions must be contiguous")
		}
	}
}
