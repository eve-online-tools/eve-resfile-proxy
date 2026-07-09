package index

import "testing"

func TestMergeWithinPlatformGlobalWins(t *testing.T) {
	osSpecific := map[string]string{
		"res:/shared.png":  "os/shared.png",
		"res:/os-only.png": "os/only.png",
	}
	global := map[string]string{
		"res:/shared.png":      "global/shared.png",
		"res:/global-only.png": "global/only.png",
	}

	merged := MergeWithinPlatform(osSpecific, global)
	if merged["res:/shared.png"] != "global/shared.png" {
		t.Fatalf("shared = %q", merged["res:/shared.png"])
	}
	if merged["res:/os-only.png"] != "os/only.png" {
		t.Fatalf("os-only = %q", merged["res:/os-only.png"])
	}
	if merged["res:/global-only.png"] != "global/only.png" {
		t.Fatalf("global-only = %q", merged["res:/global-only.png"])
	}
}

func TestMergeWithinPlatformGlobalOnly(t *testing.T) {
	global := map[string]string{"res:/a.png": "a.png"}
	merged := MergeWithinPlatform(nil, global)
	if len(merged) != 1 || merged["res:/a.png"] != "a.png" {
		t.Fatalf("merged = %#v", merged)
	}
}
