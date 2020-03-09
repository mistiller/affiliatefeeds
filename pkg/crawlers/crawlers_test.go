// +build unit
// +build !integration

package crawlers

import (
	"testing"

	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

func TestCrawler(t *testing.T) {
	if !helpers.IsOnline("") {
		t.Fatal("Currently offline")
	}

	var feeds = []CrawlFeed{
		/*CrawlFeed{
			Name:   "Boozt Bestseller Women",
			Domain: "https://www.boozt.com/se/sv/barn/nyheter?page=1&limit=200&grid=5&order=pace_asc",
			Scraper: BooztScraper{
				language:         "sv",
				nameSelector:     ".product-details__p-name",
				brandSelector:    ".product-details > h2:nth-child(1) > a:nth-child(1)",
				reviewSelector:   ".inner-wrap > ul:nth-child(1)",
				colorSelector:    ".eanColors > ul:nth-child(1)",
				categorySelector: ".product-box-splash",
			},
		},*/
		CrawlFeed{
			Name:   "Zalando Featured Women",
			Domain: "https://www.zalando.se/damklader/",
			Scraper: ZalandoScraper{
				language:      "sv",
				nameSelector:  "h1.h-text",
				brandSelector: "div.h-m-bottom-m:nth-child(2)",
				colorSelector: "div.h-m-top-s",
			},
		},
	}

	for n := range feeds {
		products, err := feeds[n].Get(false)
		if err != nil {
			t.Fatalf("%v", err)
		}

		for _, p := range products {
			if p.WebsiteFeatures == 0 || p.Name == "" {
				t.Fatalf("Product inconsistent")
			}
		}
	}
}
