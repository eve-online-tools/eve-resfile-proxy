package index

import (
	"fmt"
	"strings"
)

type Platform string

const (
	PlatformWindows Platform = "windows"
	PlatformMacOS   Platform = "macos"
)

type IndexPaths struct {
	Global     string
	OSSpecific string
}

var PlatformIndexPaths = map[Platform]IndexPaths{
	PlatformWindows: {
		Global:     "app:/resfileindex.txt",
		OSSpecific: "app:/resfileindex_Windows.txt",
	},
	PlatformMacOS: {
		Global:     "app:/EVE.app/Contents/Resources/build/resfileindex.txt",
		OSSpecific: "app:/EVE.app/Contents/Resources/build/resfileindex_macOS.txt",
	},
}

var AllPlatforms = []Platform{PlatformWindows, PlatformMacOS}

func BuildIndexURL(origin, build string, p Platform) string {
	base := strings.TrimRight(origin, "/")
	switch p {
	case PlatformWindows:
		return fmt.Sprintf("%s/eveonline_%s.txt", base, build)
	case PlatformMacOS:
		return fmt.Sprintf("%s/eveonlinemacOS_%s.txt", base, build)
	default:
		panic(fmt.Sprintf("unknown platform %q", p))
	}
}

func ParsePlatform(s string) (Platform, error) {
	if s == "" {
		return "", nil
	}
	switch strings.ToLower(s) {
	case string(PlatformWindows):
		return PlatformWindows, nil
	case string(PlatformMacOS):
		return PlatformMacOS, nil
	default:
		return "", fmt.Errorf("invalid platform %q (expected windows or macos)", s)
	}
}

func PlatformsToLoad(platform Platform) []Platform {
	if platform != "" {
		return []Platform{platform}
	}
	return AllPlatforms
}

func preferenceOrder(pref Platform) []Platform {
	if pref == PlatformMacOS {
		return []Platform{PlatformMacOS, PlatformWindows}
	}
	return []Platform{PlatformWindows, PlatformMacOS}
}
