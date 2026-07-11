package platform

import (
	"fmt"
	"slices"
	"strings"
)

type Platform string

const (
	Windows Platform = "windows"
	Mac     Platform = "macOS"
)

func (p Platform) String() string {
	return string(p)
}

func Parse(s string) (Platform, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "windows":
		return Windows, nil
	case "macos":
		return Mac, nil
	default:
		return "", fmt.Errorf("unknown platform %q", s)
	}
}

func ParseList(s string) ([]Platform, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	platforms := make([]Platform, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p, err := Parse(part)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, p)
	}
	return platforms, nil
}

// Intersect returns platforms present in every set, preserving order from the first set.
func Intersect(sets ...[]Platform) []Platform {
	if len(sets) == 0 {
		return nil
	}

	intersection := slices.Clone(sets[0])
	for _, set := range sets[1:] {
		inSet := make(map[Platform]struct{}, len(set))
		for _, p := range set {
			inSet[p] = struct{}{}
		}

		filtered := intersection[:0]
		for _, p := range intersection {
			if _, ok := inSet[p]; ok {
				filtered = append(filtered, p)
			}
		}
		intersection = filtered
		if len(intersection) == 0 {
			return nil
		}
	}

	return intersection
}

func (p Platform) ManifestPath(buildNumber string) string {
	switch p {
	case Mac:
		return fmt.Sprintf("eveonlinemacOS_%s.txt", buildNumber)
	default:
		return fmt.Sprintf("eveonline_%s.txt", buildNumber)
	}
}
