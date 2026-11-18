package implproject

import "testing"

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: " Production API ", want: "production-api"},
		{input: "payments__service", want: "payments-service"},
		{input: "A---B", want: "a-b"},
		{input: "  ", want: ""},
		{input: "v2.0/API", want: "v20api"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeSlug(tt.input); got != tt.want {
				t.Fatalf("normalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
