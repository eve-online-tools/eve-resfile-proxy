package rewrite

import (
	"fmt"
	"path"
	"sort"
	"strings"

	vfspath "github.com/eve-online-tools/eve-resfile-proxy/vfs/path"
)

// Rule maps paths under From onto paths under To. Matching is prefix-based;
// the longest From wins when multiple rules match.
type Rule struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type compiledRule struct {
	from string
	to   string
	dir  bool
}

func compileRules(rules []Rule) ([]compiledRule, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	compiled := make([]compiledRule, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))

	for i, rule := range rules {
		fromSlash := strings.HasSuffix(strings.TrimSpace(rule.From), "/")
		toSlash := strings.HasSuffix(strings.TrimSpace(rule.To), "/")
		if fromSlash != toSlash {
			return nil, fmt.Errorf("rewrites[%d]: directory aliases require trailing slashes on both from and to", i)
		}

		from, err := normalizeRulePath(rule.From)
		if err != nil {
			return nil, fmt.Errorf("rewrites[%d].from: %w", i, err)
		}
		to, err := normalizeRulePath(rule.To)
		if err != nil {
			return nil, fmt.Errorf("rewrites[%d].to: %w", i, err)
		}
		if from == to {
			return nil, fmt.Errorf("rewrites[%d]: from and to must differ", i)
		}
		if _, exists := seen[from]; exists {
			return nil, fmt.Errorf("rewrites[%d]: duplicate from path %q", i, from)
		}
		seen[from] = struct{}{}
		compiled = append(compiled, compiledRule{from: from, to: to, dir: fromSlash})
	}

	sort.Slice(compiled, func(i, j int) bool {
		return len(compiled[i].from) > len(compiled[j].from)
	})

	return compiled, nil
}

func normalizeRulePath(name string) (string, error) {
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

func (r compiledRule) translate(name string) (string, bool) {
	if name == r.from {
		return r.to, true
	}
	if strings.HasPrefix(name, r.from+"/") {
		rest := strings.TrimPrefix(name, r.from+"/")
		return path.Join(r.to, rest), true
	}
	return "", false
}

func translatePath(name string, rules []compiledRule) (string, bool) {
	for _, rule := range rules {
		if target, ok := rule.translate(name); ok {
			return target, true
		}
	}
	return "", false
}
