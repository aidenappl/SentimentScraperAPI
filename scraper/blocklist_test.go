package scraper

import (
	"slices"
	"strings"
	"testing"
)

func TestIsBlocked(t *testing.T) {
	original := BlockedDomains
	t.Cleanup(func() { BlockedDomains = original })
	SetBlockedDomains([]string{"reuters.com", "barrons.com"})

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"bare domain", "https://reuters.com/world/story", true},
		{"www subdomain", "https://www.reuters.com/world/story", true},
		{"deeper subdomain", "https://edition.reuters.com/world/story", true},
		{"second entry", "https://www.barrons.com/articles/x?mod=y", true},
		{"unblocked outlet", "https://www.cnbc.com/2026/08/09/story.html", false},
		// Must not block a different domain that merely ends with the same text.
		{"lookalike domain", "https://notreuters.com/story", false},
		{"reuters in path only", "https://example.com/reuters.com/story", false},
		{"unparseable", "://nope", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBlocked(tt.url); got != tt.want {
				t.Fatalf("IsBlocked(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestSetBlockedDomainsNormalises(t *testing.T) {
	original := BlockedDomains
	t.Cleanup(func() { BlockedDomains = original })

	SetBlockedDomains([]string{" WWW.Reuters.com ", "", "  barrons.com"})

	if !slices.Equal(BlockedDomains, []string{"reuters.com", "barrons.com"}) {
		t.Fatalf("got %v, want [reuters.com barrons.com]", BlockedDomains)
	}
	if !IsBlocked("https://www.reuters.com/x") {
		t.Fatal("normalised entry should still match")
	}
}

func TestSetBlockedDomainsCanDisable(t *testing.T) {
	original := BlockedDomains
	t.Cleanup(func() { BlockedDomains = original })

	SetBlockedDomains(nil)

	if IsBlocked("https://www.reuters.com/x") {
		t.Fatal("an empty blocklist must block nothing")
	}

	// Clearing the domain blocklist must not disable the non-HTML exclusions,
	// which are about file type rather than outlet policy.
	for _, p := range ExcludedURLPatterns() {
		if strings.Contains(p, "reuters") {
			t.Fatalf("domain pattern %q survived a cleared blocklist", p)
		}
	}
	if !IsNonHTML("https://example.com/report.pdf") {
		t.Fatal("non-HTML detection must be independent of the domain blocklist")
	}
}

func TestIsNonHTML(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"pdf", "https://www.oaktreecapital.com/docs/press/launch.pdf", true},
		{"uppercase extension", "https://example.com/Report.PDF", true},
		{"pdf with query string", "https://example.com/doc.pdf?download=1", true},
		{"spreadsheet", "https://example.com/data.xlsx", true},
		{"image", "https://example.com/chart.png", true},
		{"html article", "https://www.cnbc.com/2026/08/09/story.html", false},
		{"extensionless article", "https://techcrunch.com/2026/08/08/story/", false},
		// A path segment that merely contains the extension text is not a file.
		{"pdf in path segment", "https://example.com/pdf-guides/story", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNonHTML(tt.url); got != tt.want {
				t.Fatalf("IsNonHTML(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestSkipCrawlCoversBothReasons(t *testing.T) {
	original := BlockedDomains
	t.Cleanup(func() { BlockedDomains = original })
	SetBlockedDomains([]string{"reuters.com"})

	if !SkipCrawl("https://www.reuters.com/story") {
		t.Error("blocked outlet should be skipped")
	}
	if !SkipCrawl("https://example.com/report.pdf") {
		t.Error("non-HTML file should be skipped")
	}
	if SkipCrawl("https://www.cnbc.com/2026/08/09/story.html") {
		t.Error("an ordinary article should not be skipped")
	}
}

func TestExcludedURLPatterns(t *testing.T) {
	original := BlockedDomains
	t.Cleanup(func() { BlockedDomains = original })
	SetBlockedDomains([]string{"reuters.com"})

	patterns := ExcludedURLPatterns()

	// Both the bare-domain and subdomain forms need covering, since real rows
	// use https://www.reuters.com/... and https://reuters.com/...
	for _, want := range []string{"%//reuters.com/%", "%.reuters.com/%"} {
		if !slices.Contains(patterns, want) {
			t.Errorf("missing pattern %q in %v", want, patterns)
		}
	}

	// Guard the SQL semantics: a domain pattern must anchor on the host, or it
	// would also exclude rows that merely mention the domain in their path.
	for _, p := range patterns {
		if strings.Contains(p, "reuters") && !strings.Contains(p, "/") {
			t.Errorf("domain pattern %q is not host-anchored", p)
		}
	}

	// File-type patterns must anchor on the end of the URL.
	for _, want := range []string{"%.pdf", "%.pdf?%"} {
		if !slices.Contains(patterns, want) {
			t.Errorf("missing pattern %q", want)
		}
	}
}
