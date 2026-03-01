package storage

import (
	"bytes"
	"strings"
	"testing"
)

func TestHTMLImportExport(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	htmlContent := `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
    <DT><H3>Folder A</H3>
    <DL><p>
        <DT><A HREF="https://a1.com">A1</A>
        <DT><A HREF="https://a2.com">A2</A>
    </DL><p>
    <DT><H3>Folder B</H3>
    <DL><p>
        <DT><A HREF="https://b1.com">B1</A>
    </DL><p>
    <DT><A HREF="https://root.com">Root</A>
</DL><p>`

	// Test Import
	err := store.ImportBookmarksFromReader(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	entries := store.GetEntries()
	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	// Verify categories (the fix we made)
	expected := map[string]string{
		"https://a1.com":   "Folder A",
		"https://a2.com":   "Folder A",
		"https://b1.com":   "Folder B",
		"https://root.com": "",
	}

	for _, e := range entries {
		if expected[e.URL] != e.Category {
			t.Errorf("URL %s: expected category %q, got %q", e.URL, expected[e.URL], e.Category)
		}
	}

	// Test Export
	var buf bytes.Buffer
	err = store.ExportBookmarksToWriter(&buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	exportedHTML := buf.String()
	if !strings.Contains(exportedHTML, "Folder A") || !strings.Contains(exportedHTML, "https://a1.com") {
		t.Error("Exported HTML missing expected content")
	}
}

func TestJSONImportExport(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_ = store.AddEntry(Entry{
		URL:   "https://test.com",
		Title: "Test",
		Tags:  []string{"tag1"},
	})

	// Export
	var buf bytes.Buffer
	err := store.ExportNativeJSONToWriter(&buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	jsonStr := buf.String()
	if !strings.Contains(jsonStr, "\"tags\": [") || !strings.Contains(jsonStr, "tag1") {
		t.Errorf("Exported JSON missing tags: %s", jsonStr)
	}

	// Import into new store
	store2, cleanup2 := setupTestStore(t)
	defer cleanup2()

	err = store2.ImportNativeJSONFromReader(&buf)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	entries := store2.GetEntries()
	if len(entries) != 1 || entries[0].URL != "https://test.com" {
		t.Error("Imported entry mismatch")
	}
	if len(entries[0].Tags) != 1 || entries[0].Tags[0] != "tag1" {
		t.Error("Imported tags mismatch")
	}
}

func TestImportCornerCases(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Malformed HTML (should not crash)
	malformedHTML := `<DL><DT><A HREF="abc">No closing tags`
	err := store.ImportBookmarksFromReader(strings.NewReader(malformedHTML))
	if err != nil {
		t.Errorf("ImportBookmarksFromReader should handle malformed HTML without error if possible: %v", err)
	}

	// 2. Empty JSON
	err = store.ImportNativeJSONFromReader(strings.NewReader("[]"))
	if err != nil {
		t.Errorf("ImportNativeJSONFromReader failed on empty array: %v", err)
	}

	// 3. Duplicate Import (should ignore duplicates)
	_ = store.AddEntry(Entry{URL: "https://dup.com", Title: "Original"})
	err = store.ImportNativeJSONFromReader(strings.NewReader(`[{"url": "https://dup.com", "title": "Duplicate"}]`))
	if err != nil {
		t.Fatal(err)
	}

	entries := store.GetEntries()
	found := 0
	for _, e := range entries {
		if e.URL == "https://dup.com" {
			found++
		}
	}
	if found != 1 {
		t.Errorf("Expected 1 entry for duplicate URL, found %d", found)
	}

	// 4. Invalid JSON
	err = store.ImportNativeJSONFromReader(strings.NewReader("{ invalid }"))
	if err == nil {
		t.Error("Expected error on invalid JSON import")
	}
}

func TestRemoveEntry(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	url := "https://to-delete.com"
	_ = store.AddEntry(Entry{URL: url, Title: "Delete Me", Tags: []string{"temp"}})

	if !store.EntryExists(url) {
		t.Fatal("Entry should exist before deletion")
	}

	err := store.RemoveEntry(url)
	if err != nil {
		t.Fatalf("RemoveEntry failed: %v", err)
	}

	if store.EntryExists(url) {
		t.Error("Entry should not exist after deletion")
	}
}

func TestUpdateEntry(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	url := "https://to-update.com"
	initial := Entry{URL: url, Title: "Original", Category: "Old", Tags: []string{"tag1"}}
	_ = store.AddEntry(initial)

	updated := Entry{URL: url, Title: "New Title", Category: "New", Tags: []string{"tag2"}}
	err := store.UpdateEntry(updated)
	if err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	entries := store.GetEntries()
	found := false
	for _, e := range entries {
		if e.URL == url {
			found = true
			if e.Title != "New Title" || e.Category != "New" || len(e.Tags) != 1 || e.Tags[0] != "tag2" {
				t.Errorf("Update metadata mismatch: %+v", e)
			}
		}
	}
	if !found {
		t.Error("Updated entry not found in list")
	}
}
