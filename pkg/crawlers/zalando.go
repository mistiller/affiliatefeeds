package crawlers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"stillgrove.com/gofeedyourself/pkg/collection"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// ZalandoScraper implements the Scraper interface specifically for the boozt website
type ZalandoScraper struct {
	language      string
	nameSelector  string
	brandSelector string
	colorSelector string
}

// Scrape implements the scraper interface; takes a link to a PDP page and returns an array of SKUs (multiple colors = multiple products)
func (z ZalandoScraper) Scrape(link string) (product []feed.Product, err error) {
	const scraperName string = "ZalandoWebsite"
	doc, err := goquery.NewDocument(link)
	if err != nil {
		return product, err
	}

	/*if strings.HasSuffix(link, ".htm") == false {
		return []Product{}, fmt.Errorf("Not a product page")
	}*/
	if strings.Index(link, "/faq/") > -1 || strings.Index(link, "/modelexikon/") > -1 {
		return product, fmt.Errorf("Not a product page")
	}

	p := struct {
		name       string
		brand      string
		color      string
		colorField string
		category   string
		sku        string
		fields     []string
	}{}

	doc.Find(z.nameSelector).Each(func(index int, item *goquery.Selection) {
		p.fields = strings.Split(collection.Sanitize(item.Text()), " - ")

		if len(p.fields) == 1 {
			p.name = p.fields[0]
		} else {
			p.name = strings.Trim(p.fields[0], " ")
			p.category = strings.Trim(p.fields[1], " ")
		}
	})

	doc.Find(z.brandSelector).Each(func(index int, item *goquery.Selection) {
		p.brand = collection.Sanitize(item.Text())
	})

	doc.Find(z.colorSelector).Each(func(index int, item *goquery.Selection) {
		p.colorField = collection.Sanitize(item.Text())
		if strings.HasPrefix(p.colorField, "Färg:") == true {
			p.color = strings.Replace(p.colorField, "Färg: ", "", -1)
		}
	})

	skuSel := "div.h-flex-no-shrink:nth-child(1) > div:nth-child(1) > div:nth-child(1) > div:nth-child(1) > div:nth-child(1) > div:nth-child(2) > div:nth-child(2) > p:nth-child(3) > span:nth-child(2)"
	doc.Find(skuSel).Each(func(index int, item *goquery.Selection) {
		p.sku = item.Text()
	})

	product = []feed.Product{
		feed.Product{
			Name:            p.name,
			Brand:           p.brand,
			Color:           p.color,
			Language:        z.language,
			SKU:             p.sku,
			WebsiteFeatures: 1,
			ProviderCategories: []feed.ProviderCategory{
				feed.ProviderCategory{
					ProviderName: scraperName,
					Name:         p.category,
				},
			},
			RetailerMap: map[uint64]struct{}{
				collection.HashKey(scraperName): struct{}{},
			},
			Retailers: []feed.Retailer{
				feed.Retailer{
					Name:      scraperName,
					IsCrawler: true,
				},
			},
			FromFeeds: []int32{int32(collection.HashKey(scraperName))},
		},
	}

	return product, nil
}
