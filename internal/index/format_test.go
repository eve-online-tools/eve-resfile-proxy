package index

import "testing"

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0"},
		{512, "512"},
		{1024, "1K"},
		{1536, "1.5K"},
		{1048576, "1M"},
		{-1, "-"},
	}

	for _, tt := range tests {
		if got := FormatFileSize(tt.size); got != tt.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestFormatCompressionPercent(t *testing.T) {
	tests := []struct {
		size, compressed int64
		want             string
	}{
		{0, 0, "-"},
		{1000, 0, "-"},
		{1000, 500, "50%"},
		{4096, 2048, "50%"},
		{1000, 1200, "0%"},
	}

	for _, tt := range tests {
		if got := FormatCompressionPercent(tt.size, tt.compressed); got != tt.want {
			t.Errorf("FormatCompressionPercent(%d, %d) = %q, want %q", tt.size, tt.compressed, got, tt.want)
		}
	}
}
