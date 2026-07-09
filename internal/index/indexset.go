package index

type IndexSet struct {
	BuildNumber     string
	PlatformMaps    map[Platform]map[string]string
	LoadedPlatforms []Platform
}

func (s *IndexSet) Lookup(resPath string, pref Platform) (cdnPath string, hitPlatform Platform, ok bool) {
	for _, p := range preferenceOrder(pref) {
		m, loaded := s.PlatformMaps[p]
		if !loaded {
			continue
		}
		if cdn, found := m[resPath]; found {
			return cdn, p, true
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
