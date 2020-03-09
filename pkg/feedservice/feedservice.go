package feedservice

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	"stillgrove.com/gofeedyourself/pkg/storefront"

	awin "stillgrove.com/gofeedyourself/pkg/awin"
	crawlers "stillgrove.com/gofeedyourself/pkg/crawlers"
	cfg "stillgrove.com/gofeedyourself/pkg/feedservice/config"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/sftp"
	td "stillgrove.com/gofeedyourself/pkg/tradedoubler"
	woo "stillgrove.com/gofeedyourself/pkg/woocommerce"
)

const (
	// Retries lists the number of times the FeedService as a whole can get retried
	Retries = 2
)

var (
	// ImplementedBackends lists the eligible backend options
	ImplementedBackends = [...]string{
		"woocommerce",
		"vsf-dump",
		"csv",
	}
)

// FeedService is the central process that brings together the steps in collating and uploading the product feeds
type FeedService struct {
	mux            *sync.Mutex
	errs           PipelineErrors
	productionFlag bool
	cfg            *cfg.File
	backend        string
	doUpdate       bool
}

// New initializes and returns a FeedService pointer
func New(cfg *cfg.File, backend string, productionFlag bool) (p *FeedService, err error) {
	p = &FeedService{
		productionFlag: productionFlag,
		mux:            new(sync.Mutex),
		cfg:            cfg,
	}

	p.errs = NewPE(
		p.mux,
		p.productionFlag,
	)

	p.backend, err = checkBackend(backend)
	if err != nil {
		return p, err
	}

	return p, nil
}

func (p *FeedService) Run(applyUpdate bool, purgeImages bool) {
	defer track(time.Now(), "FeedService")

	var (
		err          error
		loc          string
		lang         string
		cc           string
		convTable    string
		dynamoID     string
		dynamoSecret string

		mapNames = [...]string{
			"colors",
			"patterns",
			"sizes",
			"genders",
		}
	)
	doUpdate := applyUpdate || p.productionFlag

	log.WithFields(
		log.Fields{
			"Started at":      time.Now().UTC(),
			"Production Flag": p.productionFlag,
		},
	).Println("FeedService Started")

	_, err = checkBackend(p.backend)
	if err != nil {
		p.errs.Log(fmt.Errorf("Incorrect backend specified - %s", p.backend), "Check Backend setting")
	}

	if !helpers.IsOnline("") {
		p.errs.Log(fmt.Errorf("No internet connection detected"), "Check Connection")
	}

	cc, loc, lang, err = p.cfg.GetLocale()
	p.errs.Log(err, "Load Country/Locale/Language from Config")

	locale, err := feed.NewLocale(cc, lang, loc)
	p.errs.Log(err, "Parse Locale from Config")

	convTable, website, err := p.cfg.GetTD()
	p.errs.Log(err, "Load TD config")

	awinAPIToken, awinFeedToken, err := p.cfg.GetAwin()
	p.errs.Log(err, "Load Awin config")

	dynamoID, dynamoSecret, _, err = p.cfg.GetDynamo()
	p.errs.Log(err, "Load Dynamo Config")

	maps := make([]map[string][]*string, len(mapNames))
	for i := range mapNames {
		maps[i], err = p.cfg.GetMapping(mapNames[i])
		p.errs.Log(err, fmt.Sprintf("Load %s Mapping", mapNames[i]))
	}

	catMap, catNameMap, err := p.cfg.GetCategoryMaps()
	p.errs.Log(err, "Get Category Map")

	td, err := td.NewFeed(
		locale,
		website.Token,
		dynamoID,
		dynamoSecret,
		convTable,
		maps[0],
		maps[1],
		maps[2],
		maps[3],
		catNameMap,
		lang,
	)
	p.errs.Log(err, "Initialize Tradedoubler Connection")

	aw, err := awin.NewAwin(
		locale,
		awinAPIToken,
		awinFeedToken,
		&feed.Mapping{
			ColorMap:   maps[0],
			SizeMap:    maps[1],
			GenderMap:  maps[2],
			PatternMap: maps[3],
			CatNameMap: catNameMap,
		},
	)

	q := feed.NewQueueFromFeeds(
		[]feed.Feed{
			td,
			aw,
		},
		p.productionFlag,
	)
	// IMPORTANT: Disabling Crawlers for now, just slowing down the testing, haven't worked out proper way to get SKU / GTin
	if false { // p.productionFlag == true {
		q.AppendMany(crawlers.GetCrawlFeeds())
	}

	if p.backend == "woocommerce" {
		domain, key, secret, err := p.cfg.GetWoo()
		p.errs.Log(err, "Load WC Config")

		w, err := woo.NewWooConnection(domain, key, secret, loc)
		p.errs.Log(err, "Initialize WC Connection")
		newestProducts := new(feed.ProductMap)
		for r := 0; r < Retries; r++ {
			newestProducts, err = q.GetPM(true)
			if err != nil {
				log.Printf("Loading products - %v", err)
				continue
			}

			p.errs.Log(err, "Load Products")

			np, nf, nc := newestProducts.Stats()
			log.Printf("Fetched %d products from %d feeds and sources with %d categories\n", np, nf, nc)

			err = w.PrepareUpdate(newestProducts, catMap, p.productionFlag, purgeImages)
			if err != nil {
				log.Printf("Failed to prepare update - %v", err)
				continue
			}
			newestProducts = nil

			output := "json"
			if doUpdate {
				output = "api"
			}

			if p.productionFlag {
				err = w.ApplyUpdate("delete", output)
				if err != nil {
					log.WithField("Error", err).Errorln("Failed to delete products")
					continue
				}

				if purgeImages {
					err = p.PurgeImages()
					if err != nil {
						log.WithField("Error", err).Errorln("Failed to purge images from FTP")
					}
				}
			}

			err = w.ApplyUpdate("createupdate", output)
			if err == nil {
				log.WithField("Queue", "createupdate").Infoln("Succeeded")
				break
			}
			purgeImages = false
			log.WithFields(
				log.Fields{
					"Queue": "createupdate",
					"Error": err,
				},
			).Errorln("Failed")
		}
		p.errs.Log(err, "Process feeds")
	} else if p.backend == "vsf-dump" {
		newestProducts := new(feed.ProductMap)
		for r := 0; r < Retries; r++ {
			newestProducts, err = q.GetPM(true)
			if err != nil {
				log.Printf("Loading products - %v", err)
				continue
			}
			p.errs.Log(err, "Load Products")

			np, nf, nc := newestProducts.Stats()
			log.Printf("Fetched %d products from %d feeds and sources with %d categories\n", np, nf, nc)

			d, err := storefront.NewFromFeed(newestProducts)
			p.errs.Log(err, "Collated feeds to update")

			err = d.WriteFiles(
				helpers.FindFolderDir("gofeedyourself") + "/dump",
			)
			p.errs.Log(err, "Write updates to files")
		}
	} else if p.backend == "csv" {
		var done bool
		newestProducts := new(feed.ProductMap)
		for r := 0; r < Retries; r++ {
			newestProducts, err = q.GetPM(false)
			if err != nil {
				log.Printf("Loading products - %v", err)
				continue
			}
			p.errs.Log(err, "Load Products")

			log.Println(newestProducts.GetFeeds())
			log.Println(newestProducts.GetRetailers())

			err = newestProducts.DumpToCSV(
				fmt.Sprintf(
					"%s/td_dump.csv",
					helpers.FindFolderDir("gofeedyourself")+"/dump",
				),
			)

			if err != nil {
				log.Warnf("error writing csv: %v", err)
				continue
			}

			done = true
			break
		}
		if !done {
			p.errs.Log(fmt.Errorf("Ran through all the allowed retries"), "Create Product CSV Dump")
		}
	}

	if len(p.errs.Errors) > 0 {
		log.WithFields(
			log.Fields{
				"Errors":               p.errs,
				"Max Memory Allocated": p.errs.GetMaxMemory(),
			},
		).Errorln("Finished with errors")
	} else {
		log.WithFields(
			log.Fields{
				"Max Memory Allocated": p.errs.GetMaxMemory(),
			},
		).Infoln("Finished without errors")
	}
}

// PurgeProducts deletes all the products from the WooCommerce backend
func (p *FeedService) PurgeProducts() {
	defer track(time.Now(), "FeedService")

	domain, key, secret, err := p.cfg.GetWoo()
	p.errs.Log(err, "Load WC Config")

	_, locale, _, err := p.cfg.GetLocale()
	p.errs.Log(err, "Load Locale from Config")

	w, err := woo.NewWooConnection(domain, key, secret, locale)
	p.errs.Log(err, "Initialize WC Connection")

	err = w.Connection.PurgeProducts(w.Locale, true)
	p.errs.Log(err, "Purge Products")

	err = p.PurgeImages()
	p.errs.Log(err, "Remove Images from FTP")
}

func (p *FeedService) PurgeImages() error {
	/*
		select
			*
		from wp_r8iicc_postmeta
		where post_id IN (
			select distinct
				post_id
			from wp_r8iicc_postmeta
			where meta_key in ('_wp_attachment_metadata', '_wp_attached_file')
				and instr(meta_value, 'STILLGROVE-') = FALSE
				and instr(meta_value, 'SGMA-') = FALSE
				and instr(meta_value, 'assets') = FALSE
		)
	*/

	const path = "/home/stillgrove/stillgrove.com/wp-content/uploads"
	var exceptions = []string{
		"/assets/",
		"SGMA-",
		"STILLGROVE-",
	}
	host, port, user, password, err := p.cfg.GetFTP()
	if err != nil {
		return fmt.Errorf("Clearing image assets - %v", err)
	}
	sess, err := sftp.NewSession(host, user, password, port)
	if err != nil {
		return fmt.Errorf("Cleaning image assets - %v", err)
	}
	defer sess.Close()

	files, err := sess.ReadDir(path)
	if err != nil {
		return fmt.Errorf("Cleaning image assets - %v", err)
	}

	var (
		matched bool
		name    string
	)
	for i := range files {
		if i%250 == 0 {
			progressBar(i, len(files))
		}
		matched = false
		if files[i].IsDir() == true {
			continue
		}
		name = files[i].Name()
		for j := range exceptions {
			matched = strings.Contains(name, exceptions[j])
			if matched == true {
				break
			}
		}
		if matched == true {
			continue
		}
		sess.Remove(path + "/" + name)
	}

	return nil
}

func progressBar(completed, total int) {
	progress := float64(completed) / float64(total) * 100.0
	s := "["
	for pct := 0.0; pct <= 100.0; pct += 4.0 {
		if pct <= progress {
			s += "#"
		} else {
			s += "-"
		}
	}
	s += fmt.Sprintf("] %s%% completed\n", strconv.FormatFloat(progress, 'f', 2, 64))
	log.WithField("Progress", s).Infoln("Processing feed")
}

func track(start time.Time, name string) {
	elapsed := time.Since(start)

	log.WithField("time elapsed", elapsed).Info(name)
}

func checkBackend(backend string) (b string, err error) {
	for i := range ImplementedBackends {
		if backend == ImplementedBackends[i] {
			b = backend
		}
	}
	if b == "" {
		return b, fmt.Errorf("Only implemented backends as are: 'woocommerce', 'csv', and 'vsf-dump'")
	}

	return b, nil
}
