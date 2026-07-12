package alias

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestCompileExtension_validation(t *testing.T) {
	t.Parallel()

	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name    string
		aliases []Alias
		wantErr string
	}{
		{
			name: "shallow under prefix",
			aliases: []Alias{{
				Alias: ".webm", Target: ".png",
				Match: &Match{PathPrefix: "ui/icons/"},
			}},
		},
		{
			name: "recursive under prefix",
			aliases: []Alias{{
				Alias: ".webm", Target: ".png",
				Match: &Match{PathPrefix: "ui/", Recursive: boolPtr(true)},
			}},
		},
		{
			name: "global recursive",
			aliases: []Alias{{
				Alias: ".webm", Target: ".png",
				Match: &Match{},
			}},
		},
		{
			name: "same extension",
			aliases: []Alias{{
				Alias: ".webm", Target: ".webm",
				Match: &Match{PathPrefix: "ui/"},
			}},
			wantErr: "must differ",
		},
		{
			name: "missing dot",
			aliases: []Alias{{
				Alias: "webm", Target: ".png",
				Match: &Match{PathPrefix: "ui/"},
			}},
			wantErr: "must start with",
		},
		{
			name: "empty alias",
			aliases: []Alias{{
				Alias: "", Target: ".png",
				Match: &Match{PathPrefix: "ui/"},
			}},
			wantErr: "must not be empty",
		},
		{
			name: "recursive false without prefix",
			aliases: []Alias{{
				Alias: ".webm", Target: ".png",
				Match: &Match{Recursive: boolPtr(false)},
			}},
			wantErr: "requires path_prefix",
		},
		{
			name: "duplicate extension rule",
			aliases: []Alias{
				{Alias: ".webm", Target: ".png", Match: &Match{PathPrefix: "ui/"}},
				{Alias: ".webm", Target: ".jpg", Match: &Match{PathPrefix: "ui/"}},
			},
			wantErr: "duplicate",
		},
		{
			name: "invalid prefix",
			aliases: []Alias{{
				Alias: ".webm", Target: ".png",
				Match: &Match{PathPrefix: "../escape/"},
			}},
			wantErr: "invalid",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(fstest.MapFS{}, tt.aliases)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestExtension_matches(t *testing.T) {
	t.Parallel()

	boolPtr := func(v bool) *bool { return &v }

	rule, err := compileExtension(0, Alias{
		Alias: ".webm", Target: ".png",
		Match: &Match{PathPrefix: "ui/icons/"},
	})
	if err != nil {
		t.Fatalf("compileExtension: %v", err)
	}

	recursive, err := compileExtension(0, Alias{
		Alias: ".webm", Target: ".png",
		Match: &Match{PathPrefix: "ui/", Recursive: boolPtr(true)},
	})
	if err != nil {
		t.Fatalf("compileExtension recursive: %v", err)
	}

	shallowUI, err := compileExtension(0, Alias{
		Alias: ".webm", Target: ".png",
		Match: &Match{PathPrefix: "ui/"},
	})
	if err != nil {
		t.Fatalf("compileExtension shallowUI: %v", err)
	}

	tests := []struct {
		name  string
		rule  compiledExtension
		path  string
		match bool
	}{
		{name: "shallow hit", rule: rule, path: "ui/icons/foo.webm", match: true},
		{name: "shallow nested miss", rule: rule, path: "ui/icons/sub/foo.webm", match: false},
		{name: "wrong ext", rule: rule, path: "ui/icons/foo.gif", match: false},
		{name: "wrong prefix", rule: rule, path: "other/icons/foo.webm", match: false},
		{name: "recursive nested", rule: recursive, path: "ui/a/b/foo.webm", match: true},
		{name: "recursive direct", rule: recursive, path: "ui/foo.webm", match: true},
		{name: "shallow ui direct", rule: shallowUI, path: "ui/foo.webm", match: true},
		{name: "shallow ui nested miss", rule: shallowUI, path: "ui/a/b.webm", match: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.rule.matches(tt.path); got != tt.match {
				t.Fatalf("matches(%q) = %v, want %v", tt.path, got, tt.match)
			}
		})
	}
}

func TestResolvePath_iterative(t *testing.T) {
	t.Parallel()

	boolPtr := func(v bool) *bool { return &v }

	c, err := compile([]Alias{
		{Alias: "graphics/effect.vulkan/", Target: "graphics/effect.dx11/"},
		{Alias: ".gr2", Target: ".cmf", Match: &Match{PathPrefix: "graphics/effect.vulkan/"}},
		{Alias: "legacy/icons/", Target: "icons/"},
		{Alias: ".webm", Target: ".png", Match: &Match{PathPrefix: "icons/", Recursive: boolPtr(true)}},
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{
			name:   "vulkan gr2 stack",
			input:  "graphics/effect.vulkan/model.gr2",
			want:   "graphics/effect.dx11/model.cmf",
			wantOK: true,
		},
		{
			name:   "legacy webm stack",
			input:  "legacy/icons/foo.webm",
			want:   "icons/foo.png",
			wantOK: true,
		},
		{
			name:   "path only",
			input:  "legacy/icons/foo.txt",
			want:   "icons/foo.txt",
			wantOK: true,
		},
		{
			name:   "ext only",
			input:  "icons/foo.webm",
			want:   "icons/foo.png",
			wantOK: true,
		},
		{
			name:   "passthrough png",
			input:  "icons/foo.png",
			want:   "icons/foo.png",
			wantOK: false,
		},
		{
			name:   "no match",
			input:  "other/file.txt",
			want:   "other/file.txt",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := resolvePath(tt.input, c.paths, c.exts)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (got %q)", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Fatalf("resolvePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompile_mixedList(t *testing.T) {
	t.Parallel()

	c, err := compile([]Alias{
		{Alias: "ui.base64/", Target: "ui/"},
		{Alias: ".webm", Target: ".png", Match: &Match{PathPrefix: "ui/"}},
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(c.paths) != 1 {
		t.Fatalf("len(paths) = %d, want 1", len(c.paths))
	}
	if len(c.exts) != 1 {
		t.Fatalf("len(exts) = %d, want 1", len(c.exts))
	}
}

func TestCompilePath_validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		aliases []Alias
		wantErr string
	}{
		{
			name: "match on path alias",
			aliases: []Alias{{
				Alias: "a", Target: "b",
				Match: &Match{},
			}},
			wantErr: "extension",
		},
		{
			name:    "duplicate alias",
			aliases: []Alias{{Alias: "a", Target: "b"}, {Alias: "a", Target: "c"}},
			wantErr: "duplicate",
		},
		{
			name:    "same alias and target",
			aliases: []Alias{{Alias: "same", Target: "same"}},
			wantErr: "must differ",
		},
		{
			name:    "empty alias",
			aliases: []Alias{{Alias: "", Target: "b"}},
			wantErr: "must not be empty",
		},
		{
			name:    "invalid alias",
			aliases: []Alias{{Alias: "../escape", Target: "b"}},
			wantErr: "invalid",
		},
		{
			name:    "dir alias without dir target",
			aliases: []Alias{{Alias: "ui.base64/", Target: "ui"}},
			wantErr: "trailing slashes",
		},
		{
			name:    "dir target without dir alias",
			aliases: []Alias{{Alias: "ui.base64", Target: "ui/"}},
			wantErr: "trailing slashes",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(fstest.MapFS{}, tt.aliases)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}
