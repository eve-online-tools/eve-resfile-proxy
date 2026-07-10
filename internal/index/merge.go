package index

// Merge applies overlays low to high. Later layers win on key collision.
func Merge(layers ...map[string]Entry) map[string]Entry {
	result := make(map[string]Entry)
	for _, layer := range layers {
		for key, value := range layer {
			result[key] = value
		}
	}
	return result
}

// MergeWithinPlatform overlays OS-specific under global; global wins on collision.
func MergeWithinPlatform(osSpecific, global map[string]Entry) map[string]Entry {
	if len(osSpecific) == 0 {
		return global
	}
	if len(global) == 0 {
		return osSpecific
	}
	return Merge(osSpecific, global)
}
