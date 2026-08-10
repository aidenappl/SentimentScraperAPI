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
	if got := BlockedURLPatterns(); len(got) != 0 {
		t.Fatalf("got %v, want no patterns", got)
	}
}

func TestBlockedURLPatterns(t *testing.T) {
	original := BlockedDomains
	t.Cleanup(func() { BlockedDomains = original })
	SetBlockedDomains([]string{"reuters.com"})

	patterns := BlockedURLPatterns()

	// Both the bare-domain and subdomain forms need covering, since real rows
	// use https://www.reuters.com/... and https://reuters.com/...
	for _, want := range []string{"%//reuters.com/%", "%.reuters.com/%"} {
		if !slices.Contains(patterns, want) {
			t.Errorf("missing pattern %q in %v", want, patterns)
		}
	}

	// Guard the SQL semantics: a LIKE pattern must anchor on the host, or it
	// would also exclude rows that merely mention the domain in their path.
	for _, p := range patterns {
		if !strings.Contains(p, "/") {
			t.Errorf("pattern %q is not host-anchored", p)
		}
	}
}
