package crawlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"stillgrove.com/gofeedyourself/pkg/collection"

	"github.com/gocolly/colly"
	"stillgrove.com/gofeedyourself/pkg/cache"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

type crawlQueue struct {
	mux      sync.RWMutex
	items    map[string]struct{}
	products []feed.Product
}

// CrawlFeed implements Feed and uses crawlQueue to scrape the featured products off of a specified domain based on field selectors
// (Not for production)
type CrawlFeed struct {
	Name    string
	Domain  string
	Scraper Scraper
	locale  *feed.Locale

	initialized bool
}

// GetCrawlFeeds returns a list of preconfigured crawlers
func GetCrawlFeeds() []feed.Feed {
	var (
		bs = BooztScraper{
			language:         "sv_se",
			nameSelector:     ".product-details__p-name",
			brandSelector:    ".product-details > h2:nth-child(1) > a:nth-child(1)",
			reviewSelector:   ".inner-wrap > ul:nth-child(1)",
			colorSelector:    ".eanColors > ul:nth-child(1)",
			categorySelector: ".product-box-splash",
		}
		zs = ZalandoScraper{
			language:      "sv_se",
			nameSelector:  "h1.h-text",
			brandSelector: "div.h-m-bottom-m:nth-child(2)",
			colorSelector: "div.h-m-top-s",
		}
		bbs = BjornBorgScraper{
			language:      "sv_se",
			nameSelector:  "h1.h-text",
			brandSelector: "div.h-m-bottom-m:nth-child(2)",
			colorSelector: "div.h-m-top-s",
		}
	)

	loc, err := feed.NewLocale("SE", "sv", "se_sv")
	if err != nil {
		return []feed.Feed{}
	}

	return []feed.Feed{
		CrawlFeed{
			Name:    "Boozt Bestseller Women",
			Domain:  "https://www.boozt.com/se/sv/barn/nyheter?page=1&limit=200&grid=5&order=pace_asc",
			Scraper: bs,
			locale:  loc,
		},
		CrawlFeed{
			Name:    "Boozt Bestseller Men",
			Domain:  "https://www.boozt.com/se/sv/klader-for-kvinnor/nyheter?page=1&limit=200&grid=5&order=pace_asc",
			Scraper: bs,
			locale:  loc,
		},
		CrawlFeed{
			Name:    "Boozt Bestseller Kids",
			Domain:  "https://www.boozt.com/se/sv/barn/view-all?page=1&limit=200&grid=5&order=pace_asc",
			Scraper: bs,
			locale:  loc,
		},
		CrawlFeed{
			Name:    "Boozt Bestseller Beauty",
			Domain:  "https://www.boozt.com/se/sv/klader-for-kvinnor/nyheter?page=1&limit=200&grid=5&order=pace_asc",
			Scraper: bs,
			locale:  loc,
		},
		CrawlFeed{
			Name:    "Zalando Featured Women",
			Domain:  "https://www.zalando.se/damklader/",
			Scraper: zs,
			locale:  loc,
		},
		CrawlFeed{
			Name:    "BjornBorg Featured Women",
			Domain:  "https://www.bjornborg.com/se/kvinna",
			Scraper: bbs,
			locale:  loc,
		},
	}
}

// GetName identifies the feed source
func (c CrawlFeed) GetName() string {
	return c.Name
}

func (c CrawlFeed) GetLocale() *feed.Locale {
	return c.locale
}

// Get array of products from a crawled website; implements Feed interface
func (c CrawlFeed) Get(productionFlag bool) (outProducts []feed.Product, err error) {
	path := helpers.FindFolderDir("gofeedyourself") + "/cache/" + c.GetName()
	badger, err := cache.NewBadgerCache(path, 3*time.Hour)
	if err != nil {
		return outProducts, fmt.Errorf("Failed to initialize crawler cache - %v", err)
	}
	defer badger.Close()

	links := getLinks(c.Domain)

	n := len(links)
	if n < 1 {
		return outProducts, fmt.Errorf("No links found")
	}

	input := make(chan string, len(links))
	workers := 16

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(input chan string) {
			defer wg.Done()

			for value := range input {
				products, err := c.Scraper.Scrape(value)
				if err != nil {
					//log.Printf("Scraping - %v", err)
					continue
				}

				payload, err := json.Marshal(products)
				if err != nil {
					//log.Printf("Scraping - %v", err)
					continue
				}
				idx := rand.Uint64()
				err = badger.Store(
					map[string][]byte{
						fmt.Sprintf("%d", idx): payload,
					},
				)
				if err != nil {
					log.WithField("Error", err).Errorln("Failed to store crawler product")
					continue
				}
			}
		}(input)
	}

	for _, job := range links {
		input <- job
	}
	close(input)

	wg.Wait()

	res, err := badger.LoadAll()
	if err != nil {
		return outProducts, fmt.Errorf("Load from cache -%v", err)
	}

	var p []feed.Product
	for i := range res {
		json.Unmarshal(res[i], &p)

		for j := range p {
			if !collection.AnyEmpty(
				[]*string{
					&p[j].Name,
					&p[j].SKU,
					&p[j].Color,
				},
			) {
				outProducts = append(outProducts, p[j])
			}
		}
		p = nil
	}

	log.Printf("Scraped %d products", len(outProducts))

	return outProducts, nil
}

func getLinks(website string) []string {
	var linkQueue = &crawlQueue{
		items: make(map[string]struct{}),
	}

	col := colly.NewCollector(
		colly.MaxDepth(1),
	)

	col.OnHTML("a", func(e *colly.HTMLElement) {
		relLink := e.Attr("href")

		absURL := strings.SplitAfter(e.Request.AbsoluteURL(relLink), "?")[0]

		if strings.HasPrefix(absURL, "itmss:") || strings.HasPrefix(absURL, "mailto:") {
			return
		}

		_, exist := linkQueue.items[absURL]
		if exist == true {
			return
		}

		linkQueue.mux.Lock()
		linkQueue.items[absURL] = struct{}{}
		linkQueue.mux.Unlock()
	})

	col.Visit(website)

	links := make([]string, len(linkQueue.items))
	var i int
	for link := range linkQueue.items {
		links[i] = link
		i++
	}

	return links
}
