package alias

import (
	"fmt"
	"path"
	"strings"

	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

type compiledPath struct {
	alias  string
	target string
	dir    bool
}

func compilePath(index int, a Alias) (compiledPath, error) {
	if a.Match != nil {
		return compiledPath{}, fmt.Errorf("aliases[%d]: path alias must not include match", index)
	}

	aliasSlash := strings.HasSuffix(strings.TrimSpace(a.Alias), "/")
	targetSlash := strings.HasSuffix(strings.TrimSpace(a.Target), "/")
	if aliasSlash != targetSlash {
		return compiledPath{}, fmt.Errorf("aliases[%d]: path alias: directory aliases require trailing slashes on both alias and target", index)
	}

	aliasPath, err := normalizePath(a.Alias)
	if err != nil {
		return compiledPath{}, fmt.Errorf("aliases[%d]: path alias: alias: %w", index, err)
	}
	targetPath, err := normalizePath(a.Target)
	if err != nil {
		return compiledPath{}, fmt.Errorf("aliases[%d]: path alias: target: %w", index, err)
	}
	if aliasPath == targetPath {
		return compiledPath{}, fmt.Errorf("aliases[%d]: path alias: alias and target must differ", index)
	}

	return compiledPath{alias: aliasPath, target: targetPath, dir: aliasSlash}, nil
}

func sortCompiledPaths(paths []compiledPath) {
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if len(paths[j].alias) > len(paths[i].alias) {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
}

func normalizePath(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	cleaned, err := vfspath.CleanFile(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid path %q", name)
	}
	return cleaned, nil
}

func (r compiledPath) translate(name string) (string, bool) {
	if name == r.alias {
		return r.target, true
	}
	if strings.HasPrefix(name, r.alias+"/") {
		rest := strings.TrimPrefix(name, r.alias+"/")
		return path.Join(r.target, rest), true
	}
	return "", false
}
