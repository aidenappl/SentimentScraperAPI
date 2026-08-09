package scraper

import (
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

// MinBodyLength is the shortest extracted body treated as a real article.
// Anything shorter is almost always a cookie wall, a paywall stub or a
// consent interstitial, and writing it back would mark the article as
// crawled when it never was.
const MinBodyLength = 200

// parser fills in article from a page body. Domain parsers handle outlets
// whose markup we know; every other domain falls through to parseGeneric,
// so no URL is ever left without a parser.
type parser func(e *colly.HTMLElement, article *ScrapedArticle)

// domainParsers maps a registrable domain suffix to its parser. Matching is
// by suffix, so "www.cnbc.com" and "cnbc.com" both resolve to the CNBC parser.
var domainParsers = map[string]parser{
	"cnbc.com":       parseCNBC,
	"reuters.com":    parseReuters,
	"techcrunch.com": parseTechCrunch,
}

// parserFor returns the parser for a host, falling back to parseGeneric.
func parserFor(host string) (p parser, named bool) {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	for domain, fn := range domainParsers {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return fn, true
		}
	}

	return parseGeneric, false
}

func parseCNBC(e *colly.HTMLElement, article *ScrapedArticle) {
	article.Title = e.ChildText("h1.ArticleHeader-headline")
	article.AuthorName = e.ChildText("a.Author-authorName, span.Author-authorInfo")

	var paragraphs []string
	e.ForEach("div.group p", func(_ int, el *colly.HTMLElement) {
		if text := strings.TrimSpace(el.Text); text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	article.ArticleBody = strings.Join(paragraphs, "\n\n")

	if article.AuthorName == "" {
		article.AuthorName = "CNBC"
	}
}

func parseReuters(e *colly.HTMLElement, article *ScrapedArticle) {
	article.Title = e.ChildText("h1")
	if category := e.ChildText("a.article-header__section"); category != "" {
		article.Category = &category
	}

	var authors []string
	e.ForEach("div[data-testid='AuthorName'] a, div[data-testid='AuthorName'] span", func(_ int, el *colly.HTMLElement) {
		name := strings.TrimSpace(el.Text)
		// Filter out the connective text between author names.
		if name != "" && name != "By" && name != "," && name != "and" {
			authors = append(authors, name)
		}
	})

	var paragraphs []string
	e.ForEach("div[data-testid^='paragraph-']", func(_ int, el *colly.HTMLElement) {
		if text := strings.TrimSpace(el.Text); text != "" {
			paragraphs = append(paragraphs, text)
		}
	})

	article.ArticleBody = strings.Join(paragraphs, "\n\n")
	article.AuthorName = strings.Join(authors, ", ")
	if article.AuthorName == "" {
		article.AuthorName = "Reuters"
	}
}

func parseTechCrunch(e *colly.HTMLElement, article *ScrapedArticle) {
	article.Title = e.ChildText("h1.article-hero__title, h1.wp-block-post-title, h1")
	article.AuthorName = e.ChildText("a.river-byline__authors, div.article-hero__authors a, a[rel='author']")

	var paragraphs []string
	e.ForEach("div.entry-content > p, div.wp-block-post-content > p, div.article-content > p", func(_ int, el *colly.HTMLElement) {
		if text := strings.TrimSpace(el.Text); text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	article.ArticleBody = strings.Join(paragraphs, "\n\n")

	if article.AuthorName == "" {
		article.AuthorName = "TechCrunch"
	}
}

// bodyCandidates are containers that hold article text on a typical news page,
// most specific first.
var bodyCandidates = []string{
	"[itemprop='articleBody']",
	"article [class*='article-body']",
	"article [class*='articleBody']",
	"div[class*='article-body']",
	"div[class*='story-body']",
	"div[class*='post-content']",
	"div[class*='entry-content']",
	"main article",
	"article",
	"main",
}

// boilerplateAncestors are containers whose paragraphs are never article text.
var boilerplateAncestors = []string{
	"nav", "header", "footer", "aside", "form", "figure", "figcaption",
	"noscript", "script", "style",
}

// parseGeneric extracts an article from an unknown outlet. It prefers
// schema.org JSON-LD, which most publishers emit and which needs no
// per-site selectors, and falls back to picking the densest block of
// paragraph text on the page.
func parseGeneric(e *colly.HTMLElement, article *ScrapedArticle) {
	if ld := parseJSONLD(e); ld != nil {
		article.Title = ld.headline
		article.AuthorName = ld.author
		article.ArticleBody = ld.body
	}

	if article.Title == "" {
		article.Title = firstNonEmpty(
			e.ChildAttr("meta[property='og:title']", "content"),
			e.ChildAttr("meta[name='twitter:title']", "content"),
			e.ChildText("h1"),
		)
	}

	if article.AuthorName == "" {
		article.AuthorName = firstNonEmpty(
			e.ChildAttr("meta[name='author']", "content"),
			e.ChildAttr("meta[property='article:author']", "content"),
			e.ChildText("a[rel='author']"),
			e.ChildText("[class*='byline'] a"),
			e.ChildText("[class*='author-name']"),
		)
	}

	if len(article.ArticleBody) < MinBodyLength {
		if body := extractDensestText(e); len(body) > len(article.ArticleBody) {
			article.ArticleBody = body
		}
	}

	if article.AuthorName == "" {
		article.AuthorName = hostLabel(e.Request.URL.Hostname())
	}
}

// extractDensestText walks the candidate containers and returns the paragraph
// text from whichever yields the most content.
func extractDensestText(e *colly.HTMLElement) string {
	best := ""

	for _, selector := range bodyCandidates {
		e.DOM.Find(selector).Each(func(_ int, sel *goquery.Selection) {
			if text := paragraphText(sel); len(text) > len(best) {
				best = text
			}
		})
		// A more specific candidate that already produced a real article
		// beats anything a broader one would find.
		if len(best) >= MinBodyLength {
			return best
		}
	}

	if best == "" {
		best = paragraphText(e.DOM.Find("body"))
	}

	return best
}

// paragraphText joins the <p> text inside sel, skipping boilerplate regions
// and one-line fragments like captions and share prompts.
func paragraphText(sel *goquery.Selection) string {
	var paragraphs []string

	sel.Find("p").Each(func(_ int, p *goquery.Selection) {
		for _, ancestor := range boilerplateAncestors {
			if p.Closest(ancestor).Length() > 0 {
				return
			}
		}

		text := strings.TrimSpace(p.Text())
		if len(text) < 40 {
			return
		}
		paragraphs = append(paragraphs, text)
	})

	return strings.Join(paragraphs, "\n\n")
}

// jsonLD holds the fields worth lifting out of a schema.org blob.
type jsonLD struct {
	headline string
	author   string
	body     string
}

// parseJSONLD scans every ld+json block on the page for an article. The
// shape varies wildly between publishers — bare object, array, or @graph —
// so the fields are pulled by walking the decoded tree rather than by
// unmarshalling into a fixed struct.
func parseJSONLD(e *colly.HTMLElement) *jsonLD {
	var found *jsonLD

	e.ForEach("script[type='application/ld+json']", func(_ int, s *colly.HTMLElement) {
		if found != nil {
			return
		}

		var doc any
		if err := json.Unmarshal([]byte(s.Text), &doc); err != nil {
			return
		}

		if node := findArticleNode(doc); node != nil {
			ld := &jsonLD{
				headline: stringField(node, "headline"),
				body:     stringField(node, "articleBody"),
				author:   authorField(node["author"]),
			}
			if ld.body != "" || ld.headline != "" {
				found = ld
			}
		}
	})

	return found
}

// findArticleNode walks a decoded JSON-LD document for the first object that
// looks like an article.
func findArticleNode(doc any) map[string]any {
	switch v := doc.(type) {
	case map[string]any:
		if _, ok := v["articleBody"]; ok {
			return v
		}
		if t := stringField(v, "@type"); strings.Contains(t, "Article") || t == "NewsArticle" || t == "BlogPosting" {
			return v
		}
		for _, nested := range v {
			if node := findArticleNode(nested); node != nil {
				return node
			}
		}
	case []any:
		for _, nested := range v {
			if node := findArticleNode(nested); node != nil {
				return node
			}
		}
	}

	return nil
}

// stringField reads a string field, tolerating the array form publishers
// sometimes emit for @type.
func stringField(node map[string]any, key string) string {
	switch v := node[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}

	return ""
}

// authorField flattens the several shapes an ld+json author takes: a string,
// an object with a name, or an array of either.
func authorField(v any) string {
	switch a := v.(type) {
	case string:
		return strings.TrimSpace(a)
	case map[string]any:
		return stringField(a, "name")
	case []any:
		var names []string
		for _, item := range a {
			if name := authorField(item); name != "" {
				names = append(names, name)
			}
		}
		return strings.Join(names, ", ")
	}

	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

// hostLabel turns a hostname into a display name for use when a page exposes
// no author at all ("www.mckinsey.com" becomes "Mckinsey").
func hostLabel(host string) string {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	if i := strings.Index(host, "."); i > 0 {
		host = host[:i]
	}
	if host == "" {
		return "Unknown"
	}

	return strings.ToUpper(host[:1]) + host[1:]
}
