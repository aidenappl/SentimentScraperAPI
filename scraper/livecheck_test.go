//go:build livecheck

// Live extraction check. Excluded from normal runs by the livecheck tag
// because it makes real requests to third-party sites.
//
// Use it when adding or debugging a parser, or when the crawl summary reports
// a persistent items_empty count and you need to know which outlets the
// extractor cannot handle:
//
//	printf '%s\n' "https://example.com/article" > /tmp/urls.txt
//	URLS_FILE=/tmp/urls.txt go test ./scraper -tags livecheck -run TestLiveExtraction -v
//
// TestLiveGenericComparison shows what a named parser contributes over the
// generic extractor for one URL, which is how you tell a parser that is
// pulling its weight from one that has quietly drifted:
//
//	COMPARE_URL="https://example.com/article" go test ./scraper -tags livecheck -run TestLiveGenericComparison -v
package scraper

import (
	"bufio"
	"os"
	"testing"
)

func TestLiveExtraction(t *testing.T) {
	requestDelay = 0

	path := os.Getenv("URLS_FILE")
	if path == "" {
		t.Skip("set URLS_FILE to a file containing one article URL per line")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		url := s.Text()
		if url == "" {
			continue
		}

		article, err := Scrape(url)
		if err != nil {
			t.Logf("FAIL  %-60.60s  %v", url, err)
			continue
		}

		t.Logf("OK    %-60.60s  chars=%d author=%q title=%.40q",
			url, len(article.ArticleBody), article.AuthorName, article.Title)
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestLiveGenericComparison(t *testing.T) {
	requestDelay = 0

	url := os.Getenv("COMPARE_URL")
	if url == "" {
		t.Skip("set COMPARE_URL to the article to compare")
	}

	if named, err := Scrape(url); err != nil {
		t.Logf("named:   FAIL %v", err)
	} else {
		t.Logf("named:   chars=%d author=%q title=%q", len(named.ArticleBody), named.AuthorName, named.Title)
	}

	saved := domainParsers
	domainParsers = map[string]parser{}
	defer func() { domainParsers = saved }()

	if generic, err := Scrape(url); err != nil {
		t.Logf("generic: FAIL %v", err)
	} else {
		t.Logf("generic: chars=%d author=%q title=%q", len(generic.ArticleBody), generic.AuthorName, generic.Title)
	}
}
