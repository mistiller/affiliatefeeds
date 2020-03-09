package tradedoubler

import (
	"fmt"
	"log"

	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	gtd "stillgrove.com/gofeedyourself/pkg/tradedoubler/client"
)

const (
	// FeedSize - Number of products to be collected from each feed
	FeedSize = 50
)

// TestFeed can be used to retrieve a product list from the Tradedouble API (NOT WORKING!)
type TestFeed struct {
	CredentialFile      string
	Domain              string
	token               string
	language            string
	initialized         bool
	conversionTableName string
	ConversionMap       map[int32]*feed.Product
	feedIDs             []int32
	mapping             *feed.Mapping
}

// GetName identifies the feed source
func (td TestFeed) GetName() string {
	return "Tradedoubler Test"
}

func (td TestFeed) GetLocale() *feed.Locale {
	return &feed.Locale{}
}

// NewTestFeed returns a pointer to an initialize Feed struct
func NewTestFeed(tdToken, language string, ColorMap, PatternMap, SizeMap, GenderMap, CatNameMap map[string][]*string) (td *TestFeed, err error) {
	if !helpers.IsOnline("") {
		panic("Can't test, host appears to be offline")
	}

	td = &TestFeed{
		language: language,
		mapping: &feed.Mapping{
			ColorMap:   ColorMap,
			PatternMap: PatternMap,
			SizeMap:    SizeMap,
			GenderMap:  GenderMap,
			CatNameMap: CatNameMap,
		},
		token: tdToken,
	}

	td.initialized = true

	return td, nil
}

// Get returns a few products from the real TD feed
func (td TestFeed) Get(productionFlag bool) (outProducts []feed.Product, err error) {
	c, err := gtd.NewConnection(td.token)
	if err != nil {
		return outProducts, fmt.Errorf("Failed to initialize td connection - %v", err)
	}

	names, err := c.QueryFeeds(td.language)
	if err != nil || len(names) == 0 {
		return outProducts, fmt.Errorf("Failed to download feed info - %v", err)
	}

	var products []gtd.Product
	p := new(feed.Product)
	tp := new(Product)
	for i := range names {
		products, err = c.SampleProductsByFeed("sv", names[i].FeedID, fmt.Sprintf("pageSize=%d;", FeedSize))
		if err != nil || len(products) == 0 {
			return outProducts, fmt.Errorf("Failed to download test feed - %v", err)
		}

		for j := range products {
			tp = &Product{
				products[j],
				td.mapping,
			}
			if err = tp.Validate(); err != nil {
				return outProducts, err
			}

			p, err = tp.ToFeedProduct()
			if err != nil {
				log.Println(err)
				continue
			}
			if !p.Active {
				continue
			}
			outProducts = append(
				outProducts,
				*p,
			)
		}
	}

	return outProducts, nil
}
