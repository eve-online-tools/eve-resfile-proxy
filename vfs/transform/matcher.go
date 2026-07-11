package transform

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type matchSpec struct {
	PathPrefix string
	PathGlob   string
	Extensions []string
	Filename   string
}

func newMatcher(spec matchSpec) (*matcher, error) {
	if spec.PathGlob != "" {
		if _, err := doublestar.Match(spec.PathGlob, ""); err != nil {
			return nil, err
		}
	}
	return &matcher{spec: spec}, nil
}

type matcher struct {
	spec matchSpec
}

func (m *matcher) matches(resPath string) bool {
	if m == nil {
		return false
	}

	spec := m.spec
	if spec.Filename != "" && resPath != spec.Filename {
		return false
	}
	if spec.PathPrefix != "" && !strings.HasPrefix(resPath, spec.PathPrefix) {
		return false
	}
	if spec.PathGlob != "" {
		ok, err := doublestar.Match(spec.PathGlob, resPath)
		if err != nil || !ok {
			return false
		}
	}
	if len(spec.Extensions) > 0 {
		ext := strings.ToLower(filepath.Ext(resPath))
		if !extensionMatches(ext, spec.Extensions) {
			return false
		}
	}

	return true
}

func extensionMatches(ext string, allowed []string) bool {
	for _, candidate := range allowed {
		if strings.ToLower(candidate) == ext {
			return true
		}
	}
	return false
}
