// Package alias wraps fs.FS and applies configurable path and extension aliases.
package alias

import (
	"fmt"
	"io/fs"
)

// Alias is one entry in the aliases config list.
// Match discriminates: nil → path alias; non-nil → extension alias.
type Alias struct {
	Alias  string `yaml:"alias"`
	Target string `yaml:"target"`
	Match  *Match `yaml:"match,omitempty"`
}

// Match scopes an extension alias.
type Match struct {
	PathPrefix string `yaml:"path_prefix"`
	Recursive  *bool  `yaml:"recursive"`
}

type compiled struct {
	paths []compiledPath
	exts  []compiledExtension
}

const maxResolvePasses = 8

// New wraps fsys with aliases. Returns fsys unchanged when aliases is empty.
func New(fsys fs.FS, aliases []Alias) (fs.FS, error) {
	c, err := compile(aliases)
	if err != nil {
		return nil, err
	}
	if len(c.paths) == 0 && len(c.exts) == 0 {
		return fsys, nil
	}
	return &FS{fsys: fsys, paths: c.paths, exts: c.exts}, nil
}

func compile(aliases []Alias) (compiled, error) {
	if len(aliases) == 0 {
		return compiled{}, nil
	}

	var out compiled
	seenPaths := map[string]struct{}{}
	seenExts := map[string]struct{}{}

	for i, a := range aliases {
		if a.Match != nil {
			r, err := compileExtension(i, a)
			if err != nil {
				return compiled{}, err
			}
			key := r.prefix + "|" + fmt.Sprintf("%t", r.recursive) + "|" + r.aliasExt
			if _, exists := seenExts[key]; exists {
				return compiled{}, fmt.Errorf("aliases[%d]: extension alias: duplicate rule for prefix %q and %s", i, r.prefix, r.aliasExt)
			}
			seenExts[key] = struct{}{}
			out.exts = append(out.exts, r)
			continue
		}
		if a.Alias == "" && a.Target == "" {
			return compiled{}, fmt.Errorf("aliases[%d]: alias and target are required", i)
		}
		r, err := compilePath(i, a)
		if err != nil {
			return compiled{}, err
		}
		if _, exists := seenPaths[r.alias]; exists {
			return compiled{}, fmt.Errorf("aliases[%d]: path alias: duplicate alias path %q", i, r.alias)
		}
		seenPaths[r.alias] = struct{}{}
		out.paths = append(out.paths, r)
	}

	sortCompiledPaths(out.paths)
	sortCompiledExtensions(out.exts)
	return out, nil
}

func resolvePath(name string, paths []compiledPath, exts []compiledExtension) (string, bool) {
	changed := false
	for pass := 0; pass < maxResolvePasses; pass++ {
		applied := false
		if target, ok := applyBestExtension(name, exts); ok {
			name = target
			changed = true
			applied = true
		}
		if target, ok := applyBestPath(name, paths); ok {
			name = target
			changed = true
			applied = true
		}
		if !applied {
			break
		}
	}
	return name, changed
}

func applyBestPath(name string, paths []compiledPath) (string, bool) {
	for _, rule := range paths {
		if target, ok := rule.translate(name); ok {
			return target, true
		}
	}
	return "", false
}

func applyBestExtension(name string, exts []compiledExtension) (string, bool) {
	for _, rule := range exts {
		if target, ok := rule.translate(name); ok {
			return target, true
		}
	}
	return "", false
}
