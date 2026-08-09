package scraper

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gocolly/colly"
)

func newResponseWithStatus(status int) *colly.Response {
	return &colly.Response{StatusCode: status}
}

func TestMain(m *testing.M) {
	// The polite crawl delay would otherwise add seconds to every test.
	requestDelay = 0
	os.Exit(m.Run())
}

func TestParserForRouting(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		wantNamed bool
	}{
		{"bare domain", "cnbc.com", true},
		{"www subdomain", "www.cnbc.com", true},
		{"uppercase", "WWW.CNBC.COM", true},
		{"other subdomain", "edition.reuters.com", true},
		{"techcrunch", "techcrunch.com", true},
		{"unknown domain falls back", "mckinsey.com", false},
		{"empty host falls back", "", false},
		// Must not match on a suffix that merely ends with the same text.
		{"lookalike domain falls back", "notcnbc.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parse, named := parserFor(tt.host)
			if named != tt.wantNamed {
				t.Fatalf("got named=%v, want %v", named, tt.wantNamed)
			}
			if parse == nil {
				t.Fatal("parserFor returned a nil parser; every domain must have one")
			}
		})
	}
}

func TestHostLabel(t *testing.T) {
	tests := []struct{ host, want string }{
		{"www.mckinsey.com", "Mckinsey"},
		{"cnbc.com", "Cnbc"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		if got := hostLabel(tt.host); got != tt.want {
			t.Errorf("hostLabel(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

func TestAuthorField(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string", "Jane Doe", "Jane Doe"},
		{"object", map[string]any{"name": "Jane Doe"}, "Jane Doe"},
		{"array of objects", []any{
			map[string]any{"name": "Jane Doe"},
			map[string]any{"name": "John Roe"},
		}, "Jane Doe, John Roe"},
		{"unsupported shape", 42, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authorField(tt.in); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindArticleNodeHandlesGraph(t *testing.T) {
	doc := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{"@type": "WebSite"},
			map[string]any{"@type": "NewsArticle", "articleBody": "hello"},
		},
	}

	node := findArticleNode(doc)
	if node == nil {
		t.Fatal("expected to find the article inside @graph")
	}
	if got := stringField(node, "articleBody"); got != "hello" {
		t.Fatalf("got articleBody=%q, want 'hello'", got)
	}
}

// serve starts a test server returning the given HTML.
func serve(t *testing.T, html string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

func longBody(prefix string) string {
	return prefix + strings.Repeat(" the quick brown fox jumps over the lazy dog.", 12)
}

func TestScrapeGenericJSONLD(t *testing.T) {
	body := longBody("Reactor output climbed sharply this quarter.")
	url := serve(t, `<html><head>
	<script type="application/ld+json">
	{"@context":"https://schema.org","@type":"NewsArticle",
	 "headline":"A Headline","author":{"name":"Jane Doe"},
	 "articleBody":"`+body+`"}
	</script></head><body><p>ignored</p></body></html>`)

	article, err := Scrape(url)
	if err != nil {
		t.Fatalf("Scrape returned an error: %v", err)
	}
	if article.Title != "A Headline" {
		t.Errorf("got title %q, want 'A Headline'", article.Title)
	}
	if article.AuthorName != "Jane Doe" {
		t.Errorf("got author %q, want 'Jane Doe'", article.AuthorName)
	}
	if !strings.Contains(article.ArticleBody, "Reactor output climbed") {
		t.Errorf("body not extracted from JSON-LD, got %q", article.ArticleBody)
	}
}

func TestScrapeGenericParagraphFallback(t *testing.T) {
	// No JSON-LD: the generic parser must fall back to paragraph density and
	// ignore navigation and footer text.
	url := serve(t, `<html><head><meta name="author" content="John Roe"></head><body>
	<nav><p>`+longBody("Navigation filler.")+`</p></nav>
	<article><p>`+longBody("The first real paragraph of the story.")+`</p>
	<p>`+longBody("The second real paragraph.")+`</p></article>
	<footer><p>`+longBody("Footer boilerplate.")+`</p></footer>
	</body></html>`)

	article, err := Scrape(url)
	if err != nil {
		t.Fatalf("Scrape returned an error: %v", err)
	}
	if !strings.Contains(article.ArticleBody, "first real paragraph") {
		t.Errorf("missing article text, got %q", article.ArticleBody)
	}
	if strings.Contains(article.ArticleBody, "Navigation filler") ||
		strings.Contains(article.ArticleBody, "Footer boilerplate") {
		t.Errorf("boilerplate leaked into the body, got %q", article.ArticleBody)
	}
	if article.AuthorName != "John Roe" {
		t.Errorf("got author %q, want 'John Roe'", article.AuthorName)
	}
}

func TestScrapeShortBodyIsEmptyError(t *testing.T) {
	// A consent wall: fetched fine, but there is no article here. It must not
	// come back as a success, or it would be persisted as an empty body and
	// the article would never be retried.
	url := serve(t, `<html><body><p>Accept cookies to continue.</p></body></html>`)

	article, err := Scrape(url)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("got err=%v, want ErrEmptyBody", err)
	}
	if article != nil {
		t.Fatalf("got a non-nil article alongside an error: %+v", article)
	}
}

func TestScrapeForbiddenReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	article, err := Scrape(srv.URL)
	if err == nil {
		t.Fatal("expected an error for a 403 response")
	}
	if article != nil {
		t.Fatalf("got a non-nil article alongside an error: %+v", article)
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("got err=%v, want it classified as 'forbidden'", err)
	}
}

func TestScrapeSendsRealUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.UserAgent()
		w.Write([]byte(`<html><body><article><p>` + longBody("Body.") + `</p></article></body></html>`))
	}))
	defer srv.Close()

	if _, err := Scrape(srv.URL); err != nil {
		t.Fatalf("Scrape returned an error: %v", err)
	}

	// Regression guard: the User-Agent used to be a random token, which is
	// what got the crawler blocked.
	if !strings.HasPrefix(got, "Mozilla/5.0") {
		t.Fatalf("got User-Agent %q, want a real browser UA", got)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		err    error
		want   string
	}{
		{"forbidden", http.StatusForbidden, errors.New("Forbidden"), "forbidden"},
		{"not found", http.StatusNotFound, errors.New("Not Found"), "not_found"},
		{"rate limited", http.StatusTooManyRequests, errors.New("Too Many Requests"), "rate_limited"},
		{"server error", http.StatusBadGateway, errors.New("Bad Gateway"), "server_error"},
		{"other status", http.StatusTeapot, errors.New("teapot"), "http_418"},
		{"transport", 0, errors.New("connection refused"), "transport"},
		{"timeout", 0, errors.New("net/http: request canceled (Client.Timeout)"), "timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A nil *colly.Response is the transport case; otherwise only the
			// status code matters to the classifier.
			got := classifyError(newResponseWithStatus(tt.status), tt.err)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
