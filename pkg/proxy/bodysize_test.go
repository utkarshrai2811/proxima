package proxy

import "testing"

func TestParseBodySize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"10MB", 10 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"10KB", 10 * 1024, false},
		{"512B", 512, false},
		{"100", 100, false},
		{"8mb", 8 * 1024 * 1024, false},
		{" 2 MB ", 2 * 1024 * 1024, false},
		{"abc", 0, true},
		{"MB", 0, true},
		{"-5MB", 0, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, err := ParseBodySize(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseBodySize(%q): expected error, got nil (value %d)", tc.in, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseBodySize(%q): unexpected error: %v", tc.in, err)
			}

			if got != tc.want {
				t.Errorf("ParseBodySize(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
