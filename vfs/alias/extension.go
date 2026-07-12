package alias

import (
	"fmt"
	"strings"

	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

type compiledExtension struct {
	prefix    string
	recursive bool
	aliasExt  string
	targetExt string
}

func compileExtension(index int, a Alias) (compiledExtension, error) {
	if a.Match == nil {
		return compiledExtension{}, fmt.Errorf("aliases[%d]: extension alias requires match", index)
	}

	aliasExt, err := normalizeExtension(a.Alias)
	if err != nil {
		return compiledExtension{}, fmt.Errorf("aliases[%d]: extension alias: alias: %w", index, err)
	}
	targetExt, err := normalizeExtension(a.Target)
	if err != nil {
		return compiledExtension{}, fmt.Errorf("aliases[%d]: extension alias: target: %w", index, err)
	}
	if aliasExt == targetExt {
		return compiledExtension{}, fmt.Errorf("aliases[%d]: extension alias: alias and target must differ", index)
	}

	prefixInput := strings.TrimSpace(a.Match.PathPrefix)
	prefixInput = strings.TrimSuffix(prefixInput, "/")
	prefix, err := vfspath.CleanDirPrefix(prefixInput)
	if err != nil {
		return compiledExtension{}, fmt.Errorf("aliases[%d]: extension alias: match.path_prefix: %w", index, err)
	}

	recursive := extensionRecursiveDefault(a.Match, prefix)
	if a.Match.Recursive != nil {
		recursive = *a.Match.Recursive
	}
	if prefix == "" && !recursive {
		return compiledExtension{}, fmt.Errorf("aliases[%d]: extension alias: recursive: false requires path_prefix", index)
	}

	return compiledExtension{
		prefix:    prefix,
		recursive: recursive,
		aliasExt:  aliasExt,
		targetExt: targetExt,
	}, nil
}

func extensionRecursiveDefault(match *Match, prefix string) bool {
	if match.Recursive != nil {
		return *match.Recursive
	}
	return prefix == ""
}

func normalizeExtension(ext string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(ext))
	if trimmed == "" {
		return "", fmt.Errorf("extension must not be empty")
	}
	if !strings.HasPrefix(trimmed, ".") {
		return "", fmt.Errorf("extension %q must start with \".\"", ext)
	}
	if len(trimmed) == 1 {
		return "", fmt.Errorf("extension %q is invalid", ext)
	}
	return trimmed, nil
}

func sortCompiledExtensions(exts []compiledExtension) {
	for i := 0; i < len(exts); i++ {
		for j := i + 1; j < len(exts); j++ {
			if extensionRuleLess(exts[j], exts[i]) {
				exts[i], exts[j] = exts[j], exts[i]
			}
		}
	}
}

func extensionRuleLess(a, b compiledExtension) bool {
	if len(a.prefix) != len(b.prefix) {
		return len(a.prefix) > len(b.prefix)
	}
	if a.recursive != b.recursive {
		return !a.recursive
	}
	return a.aliasExt < b.aliasExt
}

func (r compiledExtension) matches(name string) bool {
	if r.prefix != "" && !strings.HasPrefix(name, r.prefix) {
		return false
	}
	if !strings.HasSuffix(name, r.aliasExt) {
		return false
	}
	rest := strings.TrimPrefix(strings.TrimSuffix(name, r.aliasExt), r.prefix)
	if rest == "" {
		return false
	}
	if r.recursive {
		return true
	}
	return !strings.Contains(rest, "/")
}

func (r compiledExtension) translate(name string) (string, bool) {
	if !r.matches(name) {
		return "", false
	}
	return name[:len(name)-len(r.aliasExt)] + r.targetExt, true
}

func swapExtension(name, fromExt, toExt string) string {
	if !strings.HasSuffix(name, fromExt) {
		return name
	}
	return name[:len(name)-len(fromExt)] + toExt
}
