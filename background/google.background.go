package background

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"strings"
)

type RSS struct {
	Channel struct {
		Items []Item `xml:"item"`
	} `xml:"channel"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Source      string `xml:"source"`
}

var seenItems = make(map[string]bool)

func fetchGoogleRSS() {
	log.Println("Fetching Google RSS feed...")
	feedURL := "https://news.google.com/rss/headlines/section/topic/BUSINESS?hl=en&gl=US&ceid=US:en"

	resp, err := http.Get(feedURL)
	if err != nil {
		log.Println("❌ Failed to fetch RSS:", err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("❌ Failed to read RSS response:", err)
		return
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		log.Println("❌ Failed to parse RSS XML:", err)
		return
	}

	for _, item := range rss.Channel.Items {
		id := strings.TrimSpace(item.GUID)
		if id == "" {
			id = strings.TrimSpace(item.Link)
		}

		if !seenItems[id] {
			seenItems[id] = true
			log.Printf("New item found: %s ||| %s\n", item.Title, item.Source)
		}
	}
}
