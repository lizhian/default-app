package main

import (
	"local/defaultapp/internal/association"
	"testing"
)

// Duplicate suffixes produce identical checkbox IDs; invalid categories hide
// entries from the category tabs. Validate the catalog at its boundary.
func TestCatalogEntriesAreSelectable(t *testing.T) {
	if len(catalog) < 400 || len(catalog) > 500 {
		t.Fatal("built-in catalog must contain 400 to 500 common suffixes")
	}
	supported := map[string]bool{}
	for _, category := range categories[1:] {
		supported[category] = true
	}
	seen := map[string]bool{}
	for _, kind := range catalog {
		normalized, err := association.Normalize(kind.ext)
		if err != nil || normalized != kind.ext {
			t.Errorf("invalid suffix %q", kind.ext)
		}
		if seen[normalized] {
			t.Errorf("duplicate checkbox ID for %q", kind.ext)
		}
		seen[normalized] = true
		if kind.name == "" || !supported[kind.category] {
			t.Errorf("unusable catalog entry: %+v", kind)
		}
	}
}
