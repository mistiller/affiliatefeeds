package crawlers

import (
	"github.com/PuerkitoBio/goquery"
	. "stillgrove.com/gofeedyourself/pkg/collection"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// Scraper interface collects a number of Scrape funtions that take in the URL of a PDP and return an array of found products
type Scraper interface {
	Scrape(link string) ([]feed.Product, error)
}

// GenericScraper is used whenever no special function exists for a source
type GenericScraper struct {
	nameSelector string
}

// Scrape implements the scraper interface; takes a link to a PDP page and returns an array of SKUs (multiple colors = multiple products)
func (g GenericScraper) Scrape(link string) ([]feed.Product, error) {
	var name string

	doc, err := goquery.NewDocument(link)
	if err != nil {
		return []feed.Product{}, err
	}

	doc.Find(g.nameSelector).Each(func(index int, item *goquery.Selection) {
		name = Sanitize(item.Text())
	})

	return []feed.Product{
		feed.Product{
			Name:            name,
			WebsiteFeatures: 1,
		},
	}, nil
}
