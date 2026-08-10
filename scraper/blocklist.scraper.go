package scraper

import (
	"net/url"
	"strings"
)

// BlockedDomains are outlets that cannot be crawled at all — hard paywalls
// that answer every request with 401 regardless of headers.
//
// Articles from these outlets are still ingested, so their headline, source
// and symbols remain available; they are simply never queued for crawling.
// Retrying them forever would keep the backlog permanently non-zero and make
// it useless as a health signal.
var BlockedDomains = []string{
	"reuters.com",
	"barrons.com",
	"wsj.com",
}

// SetBlockedDomains replaces the blocklist. An empty list clears it.
func SetBlockedDomains(domains []string) {
	cleaned := make([]string, 0, len(domains))
	for _, d := range domains {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			cleaned = append(cleaned, strings.TrimPrefix(d, "www."))
		}
	}
	BlockedDomains = cleaned
}

// IsBlocked reports whether a URL belongs to a blocked outlet.
func IsBlocked(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")

	for _, domain := range BlockedDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}

	return false
}

// BlockedURLPatterns renders the blocklist as SQL LIKE patterns for excluding
// blocked outlets from a query.
//
// The exclusion has to happen in SQL rather than after the fact: the crawl
// listing is ordered newest-first and capped, so filtering afterwards would
// hand back a batch made entirely of blocked rows and starve everything else.
func BlockedURLPatterns() []string {
	patterns := make([]string, 0, len(BlockedDomains)*2)
	for _, domain := range BlockedDomains {
		// Matches https://domain/... and https://any.sub.domain/...
		patterns = append(patterns, "%//"+domain+"/%", "%."+domain+"/%")
	}

	return patterns
}
