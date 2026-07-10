package index

import "testing"

func TestMergeWithinPlatformGlobalWins(t *testing.T) {
	osSpecific := map[string]Entry{
		"res:/shared.png":  {CDNPath: "os/shared.png"},
		"res:/os-only.png": {CDNPath: "os/only.png"},
	}
	global := map[string]Entry{
		"res:/shared.png":      {CDNPath: "global/shared.png"},
		"res:/global-only.png": {CDNPath: "global/only.png"},
	}

	merged := MergeWithinPlatform(osSpecific, global)
	if merged["res:/shared.png"].CDNPath != "global/shared.png" {
		t.Fatalf("shared = %q", merged["res:/shared.png"].CDNPath)
	}
	if merged["res:/os-only.png"].CDNPath != "os/only.png" {
		t.Fatalf("os-only = %q", merged["res:/os-only.png"].CDNPath)
	}
	if merged["res:/global-only.png"].CDNPath != "global/only.png" {
		t.Fatalf("global-only = %q", merged["res:/global-only.png"].CDNPath)
	}
}

func TestMergeWithinPlatformGlobalOnly(t *testing.T) {
	global := map[string]Entry{"res:/a.png": {CDNPath: "a.png"}}
	merged := MergeWithinPlatform(nil, global)
	if len(merged) != 1 || merged["res:/a.png"].CDNPath != "a.png" {
		t.Fatalf("merged = %#v", merged)
	}
}
