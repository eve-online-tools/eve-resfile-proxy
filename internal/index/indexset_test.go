package index

import "testing"

func TestIndexSetLookupCascade(t *testing.T) {
	set := &IndexSet{
		PlatformMaps: map[Platform]map[string]string{
			PlatformWindows: {
				"res:/shared.png":   "win/shared.png",
				"res:/win-only.png": "win/only.png",
			},
			PlatformMacOS: {
				"res:/shared.png":   "mac/shared.png",
				"res:/mac-only.png": "mac/only.png",
			},
		},
		LoadedPlatforms: []Platform{PlatformWindows, PlatformMacOS},
	}

	cdn, platform, ok := set.Lookup("res:/shared.png", "")
	if !ok || cdn != "win/shared.png" || platform != PlatformWindows {
		t.Fatalf("default shared: cdn=%q platform=%q ok=%v", cdn, platform, ok)
	}

	cdn, platform, ok = set.Lookup("res:/mac-only.png", "")
	if !ok || cdn != "mac/only.png" || platform != PlatformMacOS {
		t.Fatalf("default mac-only: cdn=%q platform=%q ok=%v", cdn, platform, ok)
	}

	cdn, platform, ok = set.Lookup("res:/shared.png", PlatformMacOS)
	if !ok || cdn != "mac/shared.png" || platform != PlatformMacOS {
		t.Fatalf("macos pref shared: cdn=%q platform=%q ok=%v", cdn, platform, ok)
	}

	cdn, platform, ok = set.Lookup("res:/win-only.png", PlatformMacOS)
	if !ok || cdn != "win/only.png" || platform != PlatformWindows {
		t.Fatalf("macos pref win-only fallback: cdn=%q platform=%q ok=%v", cdn, platform, ok)
	}

	if _, _, ok := set.Lookup("res:/missing.png", ""); ok {
		t.Fatal("expected missing path to fail")
	}
}
