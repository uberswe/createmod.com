package schematic

import (
	"strings"
	"testing"
)

func Test_NBTTree_Basics(t *testing.T) {
	raw, err := DecompressForTree(handmadeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	// Root listing
	page, err := NBTTreePage(raw, "", 1, 0, 100)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	if page.Node.Type != "compound" || !page.Node.HasChildren {
		t.Fatalf("root node: %+v", page.Node)
	}
	names := map[string]TreeNode{}
	for _, c := range page.Children {
		names[c.Name] = c
	}
	for _, want := range []string{"DataVersion", "size", "palette", "blocks", "entities"} {
		if _, ok := names[want]; !ok {
			t.Errorf("root missing %s (have %v)", want, page.Children)
		}
	}
	if names["DataVersion"].Value != "3955" {
		t.Errorf("DataVersion = %q", names["DataVersion"].Value)
	}
	if !names["palette"].HasChildren {
		t.Errorf("palette should have children")
	}

	// Descend into palette[0].Name via index paths
	var palettePath string
	for i, c := range page.Children {
		_ = i
		if c.Name == "palette" {
			palettePath = c.Path
		}
	}
	sub, err := NBTTreePage(raw, palettePath, 2, 0, 100)
	if err != nil {
		t.Fatalf("palette: %v", err)
	}
	if sub.Total < 1 {
		t.Fatalf("palette empty")
	}
	if !strings.Contains(sub.Children[0].DisplayPath, "palette[0]") {
		t.Errorf("display path = %s", sub.Children[0].DisplayPath)
	}
	if sub.Nested == nil {
		t.Errorf("depth=2 returned no nested pages")
	}

	// Paging: offset beyond end is safe
	far, err := NBTTreePage(raw, palettePath, 1, 9999, 100)
	if err != nil || len(far.Children) != 0 {
		t.Errorf("offset paging: %v children=%d", err, len(far.Children))
	}
}

func Test_NBTTree_SNBT_And_Search(t *testing.T) {
	raw, err := DecompressForTree(handmadeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	snbt, err := NBTNodeSNBT(raw, "", 0)
	if err != nil {
		t.Fatalf("snbt: %v", err)
	}
	if !strings.Contains(snbt, "minecraft:") || !strings.Contains(snbt, "DataVersion") {
		t.Errorf("snbt = %.120s", snbt)
	}

	hits, err := NBTTreeSearch(raw, "facing", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatalf("no hits for 'facing'")
	}
	found := false
	for _, h := range hits {
		if h.Name == "facing" && strings.Contains(h.DisplayPath, "Properties") {
			found = true
			// The hit's path must resolve back to the same node
			node, _, err := descend(raw, h.Path)
			if err != nil || node.name != "facing" {
				t.Errorf("hit path does not resolve: %v %s", err, node.name)
			}
		}
	}
	if !found {
		t.Errorf("no Properties.facing hit: %+v", hits)
	}

	// Value search finds block ids
	hits, err = NBTTreeSearch(raw, "minecraft:chest", 50)
	if err != nil || len(hits) == 0 {
		t.Errorf("value search: %v hits=%d", err, len(hits))
	}
}

func Test_NBTTree_ArrayVirtualization(t *testing.T) {
	// Litematic BlockStates is a long[]; its elements page as scalars.
	s, err := ReadStructureNBT(handmadeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	lit, err := WriteLitematic(s)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := DecompressForTree(lit)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := NBTTreeSearch(raw, "BlockStates", 10)
	if err != nil || len(hits) == 0 {
		t.Fatalf("BlockStates not found: %v", err)
	}
	page, err := NBTTreePage(raw, hits[0].Path, 1, 0, 10)
	if err != nil {
		t.Fatalf("array page: %v", err)
	}
	if page.Node.Type != "long[]" || page.Total < 1 {
		t.Errorf("array node: %+v", page.Node)
	}
	if len(page.Children) == 0 || page.Children[0].Type != "long" {
		t.Errorf("array children: %+v", page.Children)
	}
}
