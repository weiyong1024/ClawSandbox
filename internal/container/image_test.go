package container

import "testing"

func TestIsTransientBuildFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		log  string
		want bool
		err  error
	}{
		{
			name: "apt connection failed",
			log:  "E: Failed to fetch ... Connection failed [IP: 151.101.110.132 80]",
			want: true,
		},
		{
			name: "dns failure",
			log:  "Temporary failure resolving 'deb.debian.org'",
			want: true,
		},
		{
			name: "non network build error",
			log:  "COPY entrypoint.sh /entrypoint.sh failed: file not found",
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isTransientBuildFailure(tc.log, tc.err)
			if got != tc.want {
				t.Fatalf("isTransientBuildFailure() = %v, want %v", got, tc.want)
			}
		})
	}
}
