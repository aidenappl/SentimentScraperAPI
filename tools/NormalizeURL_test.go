package tools

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// The case seen in production: a backtick stored percent-encoded,
			// turning a live article into a permanent 404.
			name: "percent-encoded trailing backtick",
			in:   "https://www.aboutamazon.com/news/company-news/andy-jassy-amazon-low-prices%60",
			want: "https://www.aboutamazon.com/news/company-news/andy-jassy-amazon-low-prices",
		},
		{
			name: "literal trailing backtick",
			in:   "https://example.com/story`",
			want: "https://example.com/story",
		},
		{
			name: "surrounding whitespace",
			in:   "  https://example.com/story  ",
			want: "https://example.com/story",
		},
		{
			name: "trailing markdown bracket",
			in:   "https://example.com/story)",
			want: "https://example.com/story",
		},
		{
			name: "query string is preserved",
			in:   "https://www.barrons.com/articles/x?refsec=markets&mod=topics_markets",
			want: "https://www.barrons.com/articles/x?refsec=markets&mod=topics_markets",
		},
		{
			name: "trailing slash is meaningful and kept",
			in:   "https://www.reuters.com/world/india/story-2026-08-05/",
			want: "https://www.reuters.com/world/india/story-2026-08-05/",
		},
		{
			name: "clean url is untouched",
			in:   "https://www.cnbc.com/2026/08/09/story.html",
			want: "https://www.cnbc.com/2026/08/09/story.html",
		},
		{
			// Trimming must never leave something less usable than the input.
			name: "garbage falls back to the original",
			in:   "not a url",
			want: "not a url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeURL(tt.in); got != tt.want {
				t.Fatalf("NormalizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
