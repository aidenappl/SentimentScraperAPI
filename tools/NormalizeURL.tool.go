package tools

import (
	"net/url"
	"strings"
)

// urlJunkSuffix are characters that turn up glued to the end of feed URLs —
// markdown and backtick fragments, stray quotes and brackets, trailing
// punctuation. They are never valid at the end of an article URL.
const urlJunkSuffix = "`'\"<>()[]{}\\|^ \t\r\n,;:"

// NormalizeURL cleans a URL as it arrives from the brief feed.
//
// The feed occasionally appends stray characters — a trailing backtick that
// gets stored percent-encoded as %60, for instance — which turns a live
// article into a permanent 404 that no parser can rescue.
func NormalizeURL(raw string) string {
	cleaned := strings.TrimSpace(raw)

	// Decode a percent-encoded tail before trimming, so %60 is recognised as
	// the backtick it is.
	if decoded, err := url.PathUnescape(cleaned); err == nil {
		cleaned = decoded
	}

	cleaned = strings.TrimRight(cleaned, urlJunkSuffix)

	// Re-parse to confirm the result is still a usable absolute URL; if the
	// trimming broke it, keep the original rather than storing something worse.
	u, err := url.Parse(cleaned)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimSpace(raw)
	}

	return cleaned
}
