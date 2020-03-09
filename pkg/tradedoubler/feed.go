package tradedoubler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	//log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"

	"stillgrove.com/gofeedyourself/pkg/cache"
	dyn "stillgrove.com/gofeedyourself/pkg/dynamoConnection"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	gtd "stillgrove.com/gofeedyourself/pkg/tradedoubler/client"
)

const (
	// SampleSize describes the limit of products to download when not in production mode
	SampleSize = uint64(5000)
	// ProductionLimit is the arbitrary hard limit I am temporarily enforcing to avoid memory issues
	// 0 means: no limit
	ProductionLimit = uint64(0)
	CacheTTL        = 12 * time.Hour
)

type tdConversion struct {
	Key                 int32   `json:"key"`
	Timestamp           int64   `json:"timestamp"`
	OrderValue          float32 `json:"orderValue"`
	EventTypeID         int32   `json:"eventTypeId"`
	ProductID           int32   `json:"productId"`
	ProductName         string  `json:"productName"`
	ProductValue        float32 `json:"productValue"`
	PublisherCommission float32 `json:"publisherCommission"`
}

// Feed can be used to retrieve a product list from the Tradedouble API (NOT WORKING!)
type Feed struct {
	Domain              string
	token               string
	initialized         bool
	pageSize            uint64
	conversionTableName string
	ConversionMap       map[int32]*feed.Product
	feedIDs             []int32
	mapping             *feed.Mapping
	batchSize           uint64
	dynamoID            string
	dynamoSecret        string
	dynamoTableName     string
	language            string
	locale              *feed.Locale
}

// GetName identifies the feed source
func (td Feed) GetName() string {
	if td.locale != nil {
		return "Tradedoubler - " + td.locale.TwoLetterCode
	}
	return "Tradedoubler"
}

func (td Feed) GetLocale() *feed.Locale {
	return td.locale
}

// NewFeed returns a pointer to an initialize Feed struct
func NewFeed(locale *feed.Locale, tdToken, dynamoID, dynamoSecret, conversionTableName string, ColorMap, PatternMap, SizeMap, GenderMap, CatNameMap map[string][]*string, language string) (*Feed, error) {
	var td = Feed{
		dynamoID:        dynamoID,
		dynamoSecret:    dynamoSecret,
		dynamoTableName: conversionTableName,
		mapping: &feed.Mapping{
			ColorMap:   ColorMap,
			PatternMap: PatternMap,
			SizeMap:    SizeMap,
			GenderMap:  GenderMap,
			CatNameMap: CatNameMap,
		},
		token:     tdToken,
		batchSize: 4000,
		language:  locale.Language,
	}

	td.initialized = true

	return &td, nil
}

// Get implements the feed interface
func (td Feed) Get(productionFlag bool) (outProducts []feed.Product, err error) {
	path := helpers.FindFolderDir("gofeedyourself") + "/cache/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}

	cache, err := cache.NewBadgerCache(path+td.GetName(), CacheTTL)
	if err != nil {
		return outProducts, fmt.Errorf("Initialize Cache -%v", err)
	}
	defer cache.Close()

	if !productionFlag {
		log.Infoln("TD: Dev mode: skipping cache, downloading feeds")
		outProducts, err = td.downloadFeeds(cache, productionFlag)
		if err != nil {
			return outProducts, fmt.Errorf("No cache and failed to download feeds - %v", err)
		}

		return outProducts, nil
	}

	res, err := cache.LoadAll()
	if err != nil {
		return outProducts, fmt.Errorf("Load from cache -%v", err)
	}

	var p []feed.Product
	for i := range res {
		json.Unmarshal(res[i], &p)

		for j := range p {
			err = p[j].Validate()
			if err != nil {
				log.WithField("Error", err).Warningf("Tradedoubler - Invalid product in cache")
				continue
			}
			outProducts = append(outProducts, p[j])
		}
		p = nil
	}
	if len(outProducts) == 0 {
		log.Infoln("TD: Cache empty, downloading feeds")
		outProducts, err = td.downloadFeeds(cache, productionFlag)
		if err != nil {
			return outProducts, fmt.Errorf("No cache and failed to download feeds - %v", err)
		}
	}

	return outProducts, nil
}

func (td *Feed) downloadFeeds(cache cache.Cache, productionFlag bool) (outProducts []feed.Product, err error) {
	if td.initialized == false {
		return outProducts, fmt.Errorf("Connection not initialized")
	}

	if false { //productionFlag == true {
		if len(td.ConversionMap) == 0 {
			err := td.loadConvFromDynamo(7)
			if err != nil {
				return outProducts, fmt.Errorf("Load conversions from DynamoDB -%v", err)
			}
		}
	}

	c, err := gtd.NewConnection(td.token)
	if err != nil {
		return outProducts, fmt.Errorf("Failed to initialize td connection - %v", err)
	}

	if td.batchSize > SampleSize && !productionFlag {
		td.batchSize = SampleSize
	}

	nProducts, err := c.InitProductFactory(td.batchSize, td.language)
	if !productionFlag {
		nProducts = SampleSize
	} else {
		if ProductionLimit > 0 {
			nProducts = ProductionLimit
		}
	}
	if err != nil {
		return outProducts, fmt.Errorf("Initialize Product Factory - %v", err)
	}

	log.WithField("Product Count", nProducts).Infoln("Download prepared")

	var (
		products     []gtd.Product
		feedProducts []*feed.Product
		counter      uint64
		done         bool
	)

	p := new(feed.Product)
	tp := new(Product)

	for !done {
		products, done, err = c.ProductFactoryNext()

		if done {
			break
		}
		if err != nil {
			return outProducts, err
		}

		feedProducts = make([]*feed.Product, 0)
		for j := range products {
			tp = &Product{
				products[j],
				td.mapping,
			}
			p, err = tp.ToFeedProduct()
			if err != nil {
				log.WithFields(
					log.Fields{
						"Error": err,
					},
				).Debugln("Dropping Product")
				continue
			}
			if p.GetKey() == 0 {
				return outProducts, fmt.Errorf("Failed to prepare product for cache - %s - %v", tp.Name, p)
			}
			if !p.Active {
				continue
			}
			feedProducts = append(
				feedProducts,
				p,
			)
		}

		writeToCache(cache, feedProducts)

		counter += uint64(len(products))
		if counter >= nProducts {
			done = true
		}

		log.WithField("Downloaded", fmt.Sprintf("%d / %d", counter, nProducts)).Infoln("Download TD Products")
	}

	outProducts, err = loadFromCache(cache)
	if err != nil {
		return outProducts, err
	}

	log.WithFields(
		log.Fields{
			"Retrieved Products": len(outProducts),
			"All Products":       nProducts,
		},
	).Infoln("Downloaded")

	return outProducts, nil
}

func loadFromCache(cache cache.Cache) (outProducts []feed.Product, err error) {
	res, err := cache.LoadAll()
	if err != nil {
		return outProducts, fmt.Errorf("Load cached products - %v", err)
	}

	var prod []feed.Product
	for k := range res {
		json.Unmarshal(res[k], &prod)

		for j := range prod {
			if prod[j].GetKey() == 0 {
				return outProducts, fmt.Errorf("Tradedoubler: Empty product in cache")
			}
			err = prod[j].Validate()
			if err != nil {
				log.WithField("Error", err).Debugln("Tradedoubler: Inconsistent product in cache")
				continue
			}
			outProducts = append(outProducts, prod[j])
		}
		prod = nil
	}
	return outProducts, nil
}

func writeToCache(cache cache.Cache, feedProducts []*feed.Product) (err error) {
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
		return fmt.Errorf("Failed to store products in cache - %v", err)
	}

	return nil
}

// ------------------------------------------------------------
// -- Get Conversion Data from DynamoDB -----------------------
//-------------------------------------------------------------

func (td *Feed) loadConvFromDynamo(queryLookbackDays int64) error {
	db, err := dyn.InitDynamoConnection(td.dynamoID, td.dynamoSecret, td.conversionTableName)
	if err != nil {
		return fmt.Errorf("Open connection to dynamo db - %v", err)
	}

	if db.ActiveSession == nil {
		err := db.GetSession()
		if err != nil {
			return err
		}
	}

	svc := dynamodb.New(db.ActiveSession)
	ts := time.Now().Unix() - (86400 * queryLookbackDays)

	filt := expression.Name("timestamp").GreaterThan(expression.Value(ts))

	expr, err := expression.NewBuilder().WithFilter(filt).Build()
	if err != nil {
		return err
	}

	params := &dynamodb.ScanInput{
		TableName:              aws.String(td.conversionTableName),
		ReturnConsumedCapacity: aws.String("TOTAL"),
		ConsistentRead:         aws.Bool(true),
		//Limit: aws.Int64(50),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		//Select:	aws.String("COUNT"),
	}

	result, err := svc.Scan(params)
	if err != nil {
		return fmt.Errorf("Scan DynamoDB table - %v", err)
	}

	var conv []tdConversion
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &conv)
	if err != nil {
		return err
	}

	td.ConversionMap = make(map[int32]*feed.Product, len(conv))

	for idx := range conv {
		ID := conv[idx].ProductID
		_, exists := td.ConversionMap[ID]
		if exists == false {
			td.ConversionMap[ID] = &feed.Product{
				Name: conv[idx].ProductName,
			}
		}
		td.ConversionMap[ID].Commision7d = td.ConversionMap[ID].Commision7d + conv[idx].PublisherCommission

		switch conv[idx].EventTypeID {
		case 4:
			td.ConversionMap[ID].Leads7d++
		case 5:
			td.ConversionMap[ID].Conversions7d++
		}
	}

	return nil
}
