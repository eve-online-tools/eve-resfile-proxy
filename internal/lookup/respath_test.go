package lookup

import "testing"

func TestFromURLPath(t *testing.T) {
	if got := FromURLPath("/icons/64/Foo.PNG"); got != "res:/icons/64/foo.png" {
		t.Fatalf("got %q", got)
	}
	if got := FromURLPath("/"); got != "" {
		t.Fatalf("root got %q", got)
	}
	if got := FromURLPath("//icons//64//foo.png"); got != "res:/icons/64/foo.png" {
		t.Fatalf("cleaned got %q", got)
	}
}
