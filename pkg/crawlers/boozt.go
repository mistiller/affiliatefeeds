package crawlers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"stillgrove.com/gofeedyourself/pkg/collection"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// BooztScraper implements the Scraper interface specifically for the boozt website
type BooztScraper struct {
	nameSelector     string //".product-details__p-name"
	language         string
	brandSelector    string //".product-details > h2:nth-child(1) > a:nth-child(1)"
	reviewSelector   string //".inner-wrap > ul:nth-child(1)"
	colorSelector    string //".eanColors > ul:nth-child(1)"
	categorySelector string //".product-box-splash"
}

// Scrape implements the scraper interface; takes a link to a PDP page and returns an array of SKUs (multiple colors = multiple products)
func (b BooztScraper) Scrape(link string) (crawlProducts []feed.Product, err error) {
	const scraperName string = "BooztWebsite"

	p := struct {
		name   string
		brand  string
		sku    string
		score  int32
		colors []string
		tags   []string
	}{
		score: 1,
	}

	if strings.HasSuffix(link, "/customer-service") || strings.HasSuffix(link, "/trustpilot") {
		return crawlProducts, fmt.Errorf("Not a PDP")
	}

	doc, err := goquery.NewDocument(link)
	if err != nil {
		return crawlProducts, err
	}

	skuSel := "html.mod-js.mod-canvas.mod-touch.mod-history.mod-boxshadow.mod-csscolumns body.lang-sv.webshop-boozt.store-se.listing.listing-banner-exists.popup-open.no-scroll div#side_panel_product-information.side-panel.side-panel--product-information.side-panel--right.side-panel--padding div.side-panel__wrap div.side-panel__scroll.nano.has-scrollbar div.nano-content div.side-panel__scroll-content div.product-description div ul.tags li.product-sku-number"

	//skuSel := ".product-sku-number"
	doc.Find(skuSel).Each(func(index int, item *goquery.Selection) {
		p.sku = collection.Sanitize(item.Text())
	})

	doc.Find(b.nameSelector).Each(func(index int, item *goquery.Selection) {
		p.name = collection.Sanitize(item.Text())
	})

	doc.Find(b.brandSelector).Each(func(index int, item *goquery.Selection) {
		p.brand = collection.Sanitize(item.Text())
	})

	doc.Find(b.reviewSelector).Each(func(index int, item *goquery.Selection) {
		aORb := regexp.MustCompile("starfull")
		matches := aORb.FindAllStringIndex(item.Text(), -1)

		if len(matches) >= 4 {
			p.score++
		}
	})

	doc.Find(b.colorSelector).Each(func(index int, item *goquery.Selection) {
		item.Find("li").Each(func(index int, item *goquery.Selection) {
			col, exist := item.Attr("title")
			if exist == false {
				return
			}
			p.colors = append(p.colors, col)
		})
	})

	doc.Find(b.categorySelector).Each(func(index int, item *goquery.Selection) {
		item.Find("a").Each(func(index int, item *goquery.Selection) {
			col, exist := item.Attr("href")
			if exist == false {
				return
			}

			p.tags = append(p.tags, col)
		})
	})

	var categories []feed.ProviderCategory
	for i := range p.tags {
		if p.tags[i] == "" {
			continue
		}

		arr := strings.Split(p.tags[i], "/")

		// break up the category path and retain the two lowest levels (arbitrary)
		for j := len(arr) - 2; j < len(arr); j++ {
			categories = append(
				categories,
				feed.ProviderCategory{
					ProviderName: scraperName,
					Name:         arr[j],
				},
			)
		}
	}

	crawlProducts = make([]feed.Product, len(p.colors))
	for i := range p.colors {
		crawlProducts[i] = feed.Product{
			Name:               strings.Trim(p.name, " "),
			Brand:              strings.Trim(p.brand, " "),
			Color:              p.colors[i],
			WebsiteFeatures:    p.score,
			ProviderCategories: categories,
			SKU:                p.sku,
			Language:           b.language,
			FromFeeds:          []int32{int32(collection.HashKey(scraperName))},
		}
		crawlProducts[i].RetailerMap = map[uint64]struct{}{
			collection.HashKey(scraperName): struct{}{},
		}
		crawlProducts[i].Retailers = []feed.Retailer{
			feed.Retailer{
				Name:      scraperName,
				IsCrawler: true,
			},
		}
	}
	return crawlProducts, nil
}
