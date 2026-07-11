package index_test

import (
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/service/http/middleware/index"
)

func TestFormatFileSize(t *testing.T) {
	t.Parallel()

	if got := index.FormatFileSize(512); got != "512" {
		t.Fatalf("FormatFileSize(512) = %q", got)
	}
	if got := index.FormatFileSize(2048); got != "2K" {
		t.Fatalf("FormatFileSize(2048) = %q", got)
	}
}

func TestFormatCompressionPercent(t *testing.T) {
	t.Parallel()

	if got := index.FormatCompressionPercent(100, 50); got != "50%" {
		t.Fatalf("FormatCompressionPercent = %q", got)
	}
}
