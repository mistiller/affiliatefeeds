package awin

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	ac "stillgrove.com/gofeedyourself/pkg/awin/client"
	"stillgrove.com/gofeedyourself/pkg/cache"
	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"

	log "github.com/sirupsen/logrus"
)

const (
	// SampleSize describes the limit of products to download when not in production mode
	SampleSize = 5000
	// ProductionLimit is the arbitrary hard limit I am temporarily enforcing to avoid memory issues
	// 0 means: no limit
	ProductionLimit = 10000
)

type Feed struct {
	Client *ac.Client
	m      *feed.Mapping
	locale *feed.Locale
}

func NewAwin(locale *feed.Locale, apiToken, productToken string, mapping *feed.Mapping) (f *Feed, err error) {
	c, err := ac.New(
		locale,
		apiToken,
		productToken,
	)
	if err != nil {
		return f, fmt.Errorf("Locale not recognized - %v", err)
	}
	return &Feed{
		Client: c,
		m:      mapping,
		locale: locale,
	}, nil
}

// GetName identifies the feed source
func (f Feed) GetName() string {
	return fmt.Sprintf("Awin - %s", f.locale.TwoLetterCode)
}

func (f Feed) GetLocale() *feed.Locale {
	return f.locale
}

/*
// Cache is still not working, somehow (synchronous access complaints, deadlocks)

func (f Feed) Get(productionFlag bool) (outProducts []feed.Product, err error) {

	/*outProducts, err = loadFromCache(f.GetName())
	if err != nil {
		return outProducts, fmt.Errorf("Retrieving Products from Awin Cache - %v", err)
	}

	if len(outProducts) == 0 {
		log.Infoln("Awin: Cache empty, downloading feeds")
		outProducts, err = f.get(productionFlag)
		if err != nil {
			return outProducts, fmt.Errorf("Downloading Awin products - %v", err)
		}

		/*err = writeToCache(f.GetName(), outProducts)
		log.WithFields(
			log.Fields{
				"Feed":  "Awin",
				"Error": err,
			},
		).Warnln("Failed to write to cache")
	}

	if len(outProducts) == 0 {
		return outProducts, fmt.Errorf("Failed to download Awin products")
	}

	return outProducts, nil
}*/

func (f Feed) Get(productionFlag bool) (outProducts []feed.Product, err error) {
	var (
		nProducts int
	)

	if productionFlag {
		if ProductionLimit > 0 {
			nProducts = ProductionLimit
		} else {
			nProducts = -1
		}
	} else {
		nProducts = SampleSize
	}
	products, err := f.Client.GetProducts(nProducts)
	if err != nil {
		return outProducts, fmt.Errorf("Loading Awin Products - %v", err)
	}

	if len(products) == 0 {
		return outProducts, fmt.Errorf("No products returned")
	}

	temp := new(Product)
	for i := range products {
		temp = &Product{
			&products[i],
			f.m,
			f.locale.Locale,
		}
		p, err := temp.ToFeedProduct()
		if err != nil {
			log.WithFields(
				log.Fields{
					"Error":  err,
					"Source": "awin",
				},
			).Debugln("Dropping Product")
			continue
		}
		if p.GetKey() == 0 {
			return outProducts, fmt.Errorf("Failed to prepare product - %s - %s", products[i].ProductName, p.SKU)
		}
		if !p.Active {
			continue
		}

		outProducts = append(outProducts, *p)

		if i != 0 && (i%2000 == 0 || i == len(products)-1) {
			log.WithField("Received", fmt.Sprintf("%d / %d", i, len(products))).Infoln("Download Awin Products")
		}
	}

	if len(outProducts) == 0 {
		return outProducts, fmt.Errorf("No valid products in the feed")
	}

	return outProducts, nil
}

func newCache(name string) (c cache.Cache, err error) {
	path := helpers.FindFolderDir("gofeedyourself") + "/cache/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
	cache, err := cache.NewBadgerCache(path+name, 4*time.Hour)
	if err != nil {
		return c, fmt.Errorf("Initialize Cache -%v", err)
	}

	return cache, nil
}

func writeToCache(name string, feedProducts []feed.Product) (err error) {
	cache, err := newCache(name)
	if err != nil {
		return fmt.Errorf("Failed to initiate cache - %v", err)
	}
	payload, err := json.Marshal(feedProducts)
	if err != nil {
		return fmt.Errorf("Failed to store products in cache - %v", err)
	}
	err = cache.Store(
		map[string][]byte{
			fmt.Sprintf("%d", rand.Int63()): payload,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to store product in cache - %v", err)
	}

	return nil
}

func loadFromCache(name string) (outProducts []feed.Product, err error) {
	cache, err := newCache(name)
	if err != nil {
		return outProducts, fmt.Errorf("Preparing Awin Cache - %v", err)
	}
	defer cache.Close()
	res, err := cache.LoadAll()
	if err != nil {
		return outProducts, fmt.Errorf("Load cached products - %v", err)
	}

	var prod []feed.Product
	for k := range res {
		json.Unmarshal(res[k], &prod)

		for j := range prod {
			if prod[j].GetKey() == 0 {
				return outProducts, fmt.Errorf("Awin: Empty product in cache")
			}
			err = prod[j].Validate()
			if err != nil {
				log.WithField("Error", err).Debugln("Awin: Inconsistent product in cache")
				continue
			}
			outProducts = append(outProducts, prod[j])
		}
		prod = nil
	}

	return outProducts, nil
}
