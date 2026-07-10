package index

import "testing"

func TestDirPrefixFromURL(t *testing.T) {
	tests := []struct {
		urlPath string
		want    string
	}{
		{"/", "res:/"},
		{"/icons/64/", "res:/icons/64/"},
		{"//icons//64//", "res:/icons/64/"},
	}

	for _, tt := range tests {
		if got := DirPrefixFromURL(tt.urlPath); got != tt.want {
			t.Errorf("DirPrefixFromURL(%q) = %q, want %q", tt.urlPath, got, tt.want)
		}
	}
}

func TestListDirectoryRoot(t *testing.T) {
	set := &IndexSet{
		LoadedPlatforms: []Platform{PlatformWindows},
		PlatformMaps: map[Platform]map[string]Entry{
			PlatformWindows: {
				"res:/icons/64/icon.png": {CDNPath: "icons/icon.png", Size: 1024, CompressedSize: 512},
				"res:/audio/track.ogg":   {CDNPath: "audio/track.ogg", Size: 2048, CompressedSize: 1024},
			},
		},
	}

	entries, ok := set.ListDirectory("res:/", "")
	if !ok {
		t.Fatal("expected ok")
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if !entries[0].IsDir || entries[0].Name != "audio" {
		t.Fatalf("entries[0] = %+v, want audio dir first", entries[0])
	}
	if !entries[1].IsDir || entries[1].Name != "icons" {
		t.Fatalf("entries[1] = %+v, want icons dir second", entries[1])
	}
}

func TestListDirectoryNested(t *testing.T) {
	set := &IndexSet{
		LoadedPlatforms: []Platform{PlatformWindows},
		PlatformMaps: map[Platform]map[string]Entry{
			PlatformWindows: {
				"res:/foo/z.txt":       {CDNPath: "z.txt", Size: 100, CompressedSize: 50},
				"res:/foo/a.txt":       {CDNPath: "a.txt", Size: 200, CompressedSize: 100},
				"res:/foo/sub/x.png":   {CDNPath: "sub/x.png"},
				"res:/foo/sub/y.png":   {CDNPath: "sub/y.png"},
				"res:/foo/other/w.ogg": {CDNPath: "other/w.ogg"},
			},
		},
	}

	entries, ok := set.ListDirectory("res:/foo/", "")
	if !ok {
		t.Fatal("expected ok")
	}

	want := []ListingEntry{
		{Name: "other", IsDir: true},
		{Name: "sub", IsDir: true},
		{Name: "a.txt", IsDir: false, Size: 200, CompressedSize: 100},
		{Name: "z.txt", IsDir: false, Size: 100, CompressedSize: 50},
	}
	if len(entries) != len(want) {
		t.Fatalf("len(entries) = %d, want %d", len(entries), len(want))
	}
	for i, e := range entries {
		if e != want[i] {
			t.Fatalf("entries[%d] = %+v, want %+v", i, e, want[i])
		}
	}
}

func TestListDirectoryPlatformFilter(t *testing.T) {
	set := &IndexSet{
		LoadedPlatforms: []Platform{PlatformWindows, PlatformMacOS},
		PlatformMaps: map[Platform]map[string]Entry{
			PlatformWindows: {
				"res:/win-only.png": {CDNPath: "win.png", Size: 100, CompressedSize: 50},
			},
			PlatformMacOS: {
				"res:/mac-only.png": {CDNPath: "mac.png", Size: 200, CompressedSize: 100},
			},
		},
	}

	entries, ok := set.ListDirectory("res:/", PlatformMacOS)
	if !ok {
		t.Fatal("expected ok")
	}
	if len(entries) != 1 || entries[0].Name != "mac-only.png" || entries[0].Size != 200 {
		t.Fatalf("entries = %+v, want mac-only.png only", entries)
	}
}

func TestListDirectoryUnionAcrossPlatforms(t *testing.T) {
	set := &IndexSet{
		LoadedPlatforms: []Platform{PlatformWindows, PlatformMacOS},
		PlatformMaps: map[Platform]map[string]Entry{
			PlatformWindows: {
				"res:/shared.png":   {CDNPath: "shared-win.png"},
				"res:/win-only.png": {CDNPath: "win.png"},
			},
			PlatformMacOS: {
				"res:/shared.png":   {CDNPath: "shared-mac.png"},
				"res:/mac-only.png": {CDNPath: "mac.png"},
			},
		},
	}

	entries, ok := set.ListDirectory("res:/", "")
	if !ok {
		t.Fatal("expected ok")
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	want := []string{"mac-only.png", "shared.png", "win-only.png"}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("names[%d] = %q, want %q (all names: %v)", i, names[i], name, names)
		}
	}
}

func TestListDirectoryNoMatch(t *testing.T) {
	set := &IndexSet{
		LoadedPlatforms: []Platform{PlatformWindows},
		PlatformMaps: map[Platform]map[string]Entry{
			PlatformWindows: {
				"res:/icons/icon.png": {CDNPath: "icons/icon.png"},
			},
		},
	}

	_, ok := set.ListDirectory("res:/missing/", "")
	if ok {
		t.Fatal("expected not ok")
	}
}
