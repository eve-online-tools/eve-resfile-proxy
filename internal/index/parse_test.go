package index

import "testing"

func TestParseIndexLine(t *testing.T) {
	entry, err := ParseIndexLine("res:/icons/64/icon.png,icons/icon_123.png,abc123,4096,2048,0644")
	if err != nil {
		t.Fatalf("ParseIndexLine: %v", err)
	}
	if entry.LogicalPath != "res:/icons/64/icon.png" {
		t.Fatalf("logical path = %q", entry.LogicalPath)
	}
	if entry.CDNPath != "icons/icon_123.png" {
		t.Fatalf("cdn path = %q", entry.CDNPath)
	}
}

func TestParseResfileIndexLowercasesKeys(t *testing.T) {
	content := "res:/Icons/64/Icon.PNG,icons/icon_123.png,hash\n"
	entries, err := ParseResfileIndex(content)
	if err != nil {
		t.Fatalf("ParseResfileIndex: %v", err)
	}
	if entries["res:/icons/64/icon.png"] != "icons/icon_123.png" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestParseBuildIndexFiltersAppPaths(t *testing.T) {
	content := "" +
		"other:/file.txt,other/file.txt,hash\n" +
		"app:/resfileindex.txt,resindex/resfileindex.txt,hash\n"
	entries, err := ParseBuildIndex(content)
	if err != nil {
		t.Fatalf("ParseBuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d", len(entries))
	}
}

func TestFindBuildIndexEntry(t *testing.T) {
	entries, err := ParseBuildIndex("app:/resfileindex_Windows.txt,os/win.txt,hash\n")
	if err != nil {
		t.Fatalf("ParseBuildIndex: %v", err)
	}
	entry, ok := FindBuildIndexEntry(entries, "app:/resfileindex_Windows.txt")
	if !ok || entry.CDNPath != "os/win.txt" {
		t.Fatalf("entry = %+v ok=%v", entry, ok)
	}
}
