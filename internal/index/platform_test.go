package index

import "testing"

func TestBuildIndexURL(t *testing.T) {
	origin := "https://binaries.eveonline.com"
	if got := BuildIndexURL(origin, "123", PlatformWindows); got != "https://binaries.eveonline.com/eveonline_123.txt" {
		t.Fatalf("windows url = %q", got)
	}
	if got := BuildIndexURL(origin, "123", PlatformMacOS); got != "https://binaries.eveonline.com/eveonlinemacOS_123.txt" {
		t.Fatalf("macos url = %q", got)
	}
}

func TestParsePlatform(t *testing.T) {
	p, err := ParsePlatform("")
	if err != nil || p != "" {
		t.Fatalf("empty: p=%q err=%v", p, err)
	}
	p, err = ParsePlatform("windows")
	if err != nil || p != PlatformWindows {
		t.Fatalf("windows: p=%q err=%v", p, err)
	}
	p, err = ParsePlatform("macOS")
	if err != nil || p != PlatformMacOS {
		t.Fatalf("macos: p=%q err=%v", p, err)
	}
	if _, err := ParsePlatform("linux"); err == nil {
		t.Fatal("expected error for linux")
	}
}

func TestPlatformsToLoad(t *testing.T) {
	if len(PlatformsToLoad("")) != 2 {
		t.Fatalf("expected both platforms")
	}
	if len(PlatformsToLoad(PlatformWindows)) != 1 || PlatformsToLoad(PlatformWindows)[0] != PlatformWindows {
		t.Fatalf("expected windows only")
	}
}
