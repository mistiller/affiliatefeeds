package feedservice

import (
	"encoding/json"
	"testing"

	log "github.com/sirupsen/logrus"
	crawlers "stillgrove.com/gofeedyourself/pkg/crawlers"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/storefront"
)

func TestVSF(t *testing.T) {
	t.Skip()
	var (
		err error
	)
	q := new(feed.Queue)
	newestProducts := new(feed.ProductMap)

	testFeed := feed.NewTestFeed("test")

	q = feed.NewQueueFromFeeds([]feed.Feed{testFeed}, false)
	// IMPORTANT: Disabling Crawlers for now, just slowing down the testing, haven't worked out proper way to get SKU / GTin
	if false { // p.productionFlag == true {
		q.AppendMany(crawlers.GetCrawlFeeds())
	}

	newestProducts, err = q.GetPM(true)
	if err != nil {
		t.Fatal(err)
	}

	np, nf, nc := newestProducts.Stats()
	log.Printf("Fetched %d products from %d feeds and sources with %d categories\n", np, nf, nc)

	d, err := storefront.NewFromFeed(newestProducts)
	if err != nil {
		t.Fatal(err)
	}

	_, _, p, err := d.GetData()
	if err != nil {
		t.Fatal(err)
	}

	products := make(map[string]interface{})
	err = json.Unmarshal(p, &products)
	if err != nil {
		t.Fatal(err)
	}
	for k := range products {
		if products[k].(map[string]interface{})["sku"] == "" {
			t.Fatalf("No SKU created")
		}
		if products[k].(map[string]interface{})["category_ids"] == nil {
			t.Fatalf("No categories created - %v", products[k])
		}
	}
}
