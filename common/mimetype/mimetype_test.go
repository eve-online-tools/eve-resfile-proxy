package mimetype

import "testing"

func TestForFilenameKnownExtension(t *testing.T) {
	if got := ForFilename("icon.png"); got != "image/png" {
		t.Fatalf("ForFilename(icon.png) = %q", got)
	}
}

func TestForFilenameUnknownExtension(t *testing.T) {
	if got := ForFilename("data.bin"); got != "application/octet-stream" {
		t.Fatalf("ForFilename(data.bin) = %q", got)
	}
}

func TestForFilenameEveExtension(t *testing.T) {
	if got := ForFilename("modules.bnk"); got != "application/octet-stream" {
		t.Fatalf("ForFilename(modules.bnk) = %q", got)
	}
}

func TestForFilenameKnownStructured(t *testing.T) {
	if got := ForFilename("metadata.yaml"); got != "text/yaml" {
		t.Fatalf("ForFilename(metadata.yaml) = %q", got)
	}
}
