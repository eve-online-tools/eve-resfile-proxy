package index

type IndexSet struct {
	BuildNumber     string
	PlatformMaps    map[Platform]map[string]Entry
	LoadedPlatforms []Platform
}

func (s *IndexSet) Lookup(resPath string, pref Platform) (cdnPath string, hitPlatform Platform, ok bool) {
	for _, p := range preferenceOrder(pref) {
		m, loaded := s.PlatformMaps[p]
		if !loaded {
			continue
		}
		if entry, found := m[resPath]; found {
			return entry.CDNPath, p, true
		}
	}
	return "", "", false
}

func (s *IndexSet) EntryCount(p Platform) int {
	if m, ok := s.PlatformMaps[p]; ok {
		return len(m)
	}
	return 0
}
