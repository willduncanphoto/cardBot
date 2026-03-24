package detect

import "testing"

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{-1, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{1395864371200, "1.3 TB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsHidden(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		{".DS_Store", true},
		{"._resource", true},
		{".Trashes", true},
		{".hidden", true},
		{"DSC_0001.NEF", false},
		{"100NIKON", false},
	}
	for _, tt := range tests {
		got := IsHidden(tt.name)
		if got != tt.want {
			t.Errorf("IsHidden(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
