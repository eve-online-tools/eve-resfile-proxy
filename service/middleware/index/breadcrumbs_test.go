package index

import (
	"net/url"
	"testing"
)

func TestBuildBreadcrumbsRoot(t *testing.T) {
	crumbs := buildBreadcrumbs(mustURL("/"))
	if len(crumbs) != 1 {
		t.Fatalf("crumbs = %+v, want single root crumb", crumbs)
	}
	if !crumbs[0].Current || crumbs[0].Label != "res:" {
		t.Fatalf("root crumb = %+v", crumbs[0])
	}
}

func TestBuildBreadcrumbsNested(t *testing.T) {
	crumbs := buildBreadcrumbs(mustURL("/foo/bar/"))
	if len(crumbs) != 3 {
		t.Fatalf("crumbs = %+v, want 3", crumbs)
	}
	if crumbs[0].Label != "res:" || string(crumbs[0].Href) != "/" || crumbs[0].Current {
		t.Fatalf("root crumb = %+v", crumbs[0])
	}
	if crumbs[1].Label != "foo" || string(crumbs[1].Href) != "/foo/" || crumbs[1].Current {
		t.Fatalf("foo crumb = %+v", crumbs[1])
	}
	if crumbs[2].Label != "bar" || crumbs[2].Href != "" || !crumbs[2].Current {
		t.Fatalf("bar crumb = %+v", crumbs[2])
	}
}

func TestBuildBreadcrumbsPreservesQuery(t *testing.T) {
	u := mustURL("/foo/?platform=windows")
	crumbs := buildBreadcrumbs(u)
	if string(crumbs[0].Href) != "/?platform=windows" {
		t.Fatalf("root href = %q", crumbs[0].Href)
	}
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
