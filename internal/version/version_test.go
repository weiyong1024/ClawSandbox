package version

import "testing"

func TestImageTag(t *testing.T) {
	original := Version
	t.Cleanup(func() {
		Version = original
	})

	tests := []struct {
		version string
		want    string
	}{
		{version: "dev", want: "latest"},
		{version: "unknown", want: "latest"},
		{version: "v0.1.0", want: "v0.1.0"},
		{version: "1.2.3", want: "1.2.3"},
		{version: "v0.1.0-dirty", want: "latest"},
		{version: "v0.1.0-3-gabcdef", want: "latest"},
		{version: "v0.1.0-44-g27e904a-dirty", want: "latest"},
	}

	for _, tt := range tests {
		Version = tt.version
		if got := ImageTag(); got != tt.want {
			t.Fatalf("ImageTag() for %q = %q, want %q", tt.version, got, tt.want)
		}
	}
}
