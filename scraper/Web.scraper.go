package scraper

import (
	"strings"
	"time"

	"github.com/aidenappl/SentimentScraperAPI/tools"
	"github.com/gocolly/colly"
)

type ScrapedArticle struct {
	Title       string
	AuthorName  string
	ArticleBody string
	Category    *string
}

func Scrape(url string) *ScrapedArticle {
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (Linux; Android 11; SAMSUNG SM-G973U) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/14.2 Chrome/87.0.4280.141 Mobile Safari/537.36"),
	)

	article := &ScrapedArticle{}

	c.SetRequestTimeout(30 * time.Second)
	c.Limit(&colly.LimitRule{
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	})

	if strings.Contains(url, "cnbc.com") {

		c.OnHTML("body", func(e *colly.HTMLElement) {
			article.Title = e.ChildText("h1.ArticleHeader-headline")
			article.AuthorName = e.ChildText("a.Author-authorName")

			e.ForEach("div.group p", func(_ int, el *colly.HTMLElement) {
				text := strings.TrimSpace(el.Text)
				if text != "" {
					article.ArticleBody += text + "\n\n"
				}
			})
		})
	} else if strings.Contains(url, "reuters.com") {
		c.OnHTML("body", func(e *colly.HTMLElement) {
			title := e.ChildText("h1")
			category := e.ChildText("a.article-header__section")

			var authors []string

			e.ForEach("div[data-testid='AuthorName'] a, div[data-testid='AuthorName'] span", func(_ int, el *colly.HTMLElement) {
				name := strings.TrimSpace(el.Text)

				// Filter out non-author text like "By", ",", and "and"
				if name != "" && name != "By" && name != "," && name != "and" {
					authors = append(authors, name)
				}
			})

			var paragraphs []string
			e.ForEach("div[data-testid^='paragraph-']", func(_ int, el *colly.HTMLElement) {
				text := strings.TrimSpace(el.Text)
				if text != "" {
					paragraphs = append(paragraphs, text)
				}
			})

			article = &ScrapedArticle{
				Title:       title,
				AuthorName:  strings.Join(authors, ", "),
				Category:    tools.StringP(category),
				ArticleBody: strings.Join(paragraphs, "\n\n"),
			}
		})
	}

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("X-Requested-With", "XMLHttpRequest")
		r.Headers.Set("User-Agent", tools.RandomString())
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Referer", "https://www.google.com/")
		r.Headers.Set("Cache-Control", "no-cache")
		r.Headers.Set("Pragma", "no-cache")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
	})

	c.OnError(func(r *colly.Response, err error) {
		println("Error visiting:", r.Request.URL.String(), "Error:", err.Error())
	})

	err := c.Visit(url)
	if err != nil {
		println("❌ Visit error:", err.Error())
	}

	c.Wait()

	return article
}
