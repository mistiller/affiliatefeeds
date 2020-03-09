package crawlers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"stillgrove.com/gofeedyourself/pkg/collection"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// https://www.bjornborg.com/se/kvinna
// brandsku = .product-info-sku

// BjornBorgScraper implements the Scraper interface specifically for the boozt website
type BjornBorgScraper struct {
	language      string
	nameSelector  string
	brandSelector string
	colorSelector string
	skuSelector   string
}

// Scrape implements the scraper interface; takes a link to a PDP page and returns an array of SKUs (multiple colors = multiple products)
func (bb BjornBorgScraper) Scrape(link string) (product []feed.Product, err error) {
	const scraperName string = "BjornBorgWebsite"
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

	doc.Find(bb.nameSelector).Each(func(index int, item *goquery.Selection) {
		p.fields = strings.Split(collection.Sanitize(item.Text()), " - ")

		if len(p.fields) == 1 {
			p.name = p.fields[0]
		} else {
			p.name = strings.Trim(p.fields[0], " ")
			p.category = strings.Trim(p.fields[1], " ")
		}
	})

	doc.Find(bb.brandSelector).Each(func(index int, item *goquery.Selection) {
		p.brand = collection.Sanitize(item.Text())
	})

	doc.Find(bb.colorSelector).Each(func(index int, item *goquery.Selection) {
		p.colorField = collection.Sanitize(item.Text())
		if strings.HasPrefix(p.colorField, "Färg:") == true {
			p.color = strings.Replace(p.colorField, "Färg: ", "", -1)
		}
	})

	doc.Find(bb.skuSelector).Each(func(index int, item *goquery.Selection) {
		p.sku = item.Text()
	})

	product = []feed.Product{
		feed.Product{
			Name:            p.name,
			Brand:           p.brand,
			Color:           p.color,
			Language:        bb.language,
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

// https://www.bjornborg.com/se/kvinna
// brandsku = .product-info-sku
