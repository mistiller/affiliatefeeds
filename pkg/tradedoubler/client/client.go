package tradedoublerclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type productFactory struct {
	it          uint64
	initialized bool
	queue       []requestQueue
	queueLength uint64
	nProducts   uint64
}

//Connection is the object that carries the credentials and deals with the request queue
type Connection struct {
	token   string
	queue   requestQueue
	URL     string
	factory productFactory
}

// NewConnection returns a td connection pointer and tests the connection to the TD API
func NewConnection(token string) (c *Connection, err error) {
	c = new(Connection)
	if len(token) < 1 {
		return c, errors.New("Supplied an empty token")
	}
	c.URL = "http://api.tradedoubler.com/1.0/"

	c.token = token
	c.queue.connection = c

	return c, nil
}

// QueryProductsByFeed returns a list of all the products for a feed given the options from the query string
// http://dev.tradedoubler.com/products/publisher/#Matrix_syntax
func (c *Connection) QueryProductsByFeed(feedID uint64, queryString string) ([]Product, error) {
	var products []Product

	feedInfo, err := c.QueryFeeds("sv")
	if err != nil {
		return products, err
	}

	var np uint64
	for i := range feedInfo {
		if feedInfo[i].FeedID == feedID {
			np = feedInfo[i].NumberOfProducts
		}
	}
	pages := np / 100
	for nPage := uint64(1); nPage <= pages; nPage++ {
		endpoint := fmt.Sprintf("productsUnlimited;fid=%d;", feedID) + queryString
		endpoint += fmt.Sprintf(";page=%d", nPage)

		c.queue.pushGetRequest(endpoint)
	}

	data, _ := c.queue.execute()

	var response Feed
	for j := range data {
		err := json.Unmarshal(data[j], &response)
		if err != nil {
			return products, err
		}
		for idx := range response.Products {
			response.Products[idx].FeedID = int32(feedID)
			products = append(products, response.Products[idx])
		}
	}

	return products, nil
}

// SampleProductsByFeed returns a list of all the products for a feed given the options from the query string
// http://dev.tradedoubler.com/products/publisher/#Matrix_syntax
func (c *Connection) SampleProductsByFeed(languageCode string, feedID uint64, queryString string) (products []Product, err error) {
	endpoint := fmt.Sprintf("productsUnlimited;fid=%d;", feedID) + queryString
	endpoint += ";page=1"

	c.queue.pushGetRequest(endpoint)
	data, err := c.queue.execute()
	if err != nil {
		return products, fmt.Errorf("Sample from feed - %v", err)
	}

	var response Feed
	for j := range data {
		err := json.Unmarshal(data[j], &response)
		if err != nil {
			return products, err
		}
		for idx := range response.Products {
			response.Products[idx].FeedID = int32(feedID)
			products = append(products, response.Products[idx])
		}
	}

	return products, nil
}

// QueryAllProducts returns a list of all the products given the options from the query string
// http://dev.tradedoubler.com/products/publisher/#Matrix_syntax
func (c *Connection) QueryAllProducts(queryString, languageCode string) ([]Product, error) {
	var products []Product

	feedInfo, err := c.QueryFeeds(languageCode)
	if err != nil {
		return products, err
	}

	for ix := range feedInfo {
		pages := int(feedInfo[ix].NumberOfProducts / 100)
		for nPage := 1; nPage <= pages; nPage++ {
			endpoint := fmt.Sprintf("products;fid=%d;pageSize=100;page=%d", feedInfo[ix].FeedID, nPage) + queryString

			c.queue.pushGetRequest(endpoint)
		}

		data, err := c.queue.execute()
		if err != nil {
			return products, err
		}

		for i := range data {
			response := new(Feed)
			err := json.Unmarshal(data[i], &response)
			if err != nil {
				fmt.Println(err)
				break
			}
			for j := range response.Products {
				response.Products[j].FeedID = int32(feedInfo[ix].FeedID)
				products = append(products, response.Products[j])
			}
		}
	}

	return products, nil
}

// InitProductFactory factory returns the pointer to an iterator that returns product badges
// queryString: http://dev.tradedoubler.com/products/publisher/#Matrix_syntax
func (c *Connection) InitProductFactory(productsPerBatch uint64, language string) (nProducts uint64, err error) {
	vars := struct {
		it           uint64
		pages        uint64
		batchCounter uint64
		batchSize    uint64
		pageSize     uint64
	}{
		it:       1,
		pageSize: 100,
	}

	vars.batchSize = productsPerBatch / vars.pageSize

	if productsPerBatch < vars.pageSize {
		vars.pageSize = productsPerBatch
		vars.batchSize = 1
	}

	feedInfo, err := c.QueryFeeds(language)
	if err != nil {
		return nProducts, fmt.Errorf("Failed to load feeds - %v", err)
	}

	/*if strings.HasPrefix(queryString, ";") == false {
		queryString = fmt.Sprintf(";%s", queryString)
	}*/

	var q = requestQueue{
		connection: c,
	}

	for ix := range feedInfo {
		vars.pages = feedInfo[ix].NumberOfProducts / vars.pageSize

		c.factory.nProducts += feedInfo[ix].NumberOfProducts

		vars.batchCounter = 0
		for nPage := uint64(1); nPage <= vars.pages; nPage++ {
			if vars.batchCounter == vars.batchSize || nPage == vars.pages {
				c.factory.queue = append(c.factory.queue, q)
				q = requestQueue{
					connection: c,
				}
				vars.batchCounter = 0
			}
			q.pushGetRequest(
				fmt.Sprintf("productsUnlimited;fid=%d;pageSize=%d;page=%d", feedInfo[ix].FeedID, vars.pageSize, nPage),
			)

			vars.it++
			vars.batchCounter++
		}
	}

	if len(c.factory.queue) < 1 {
		return 0, fmt.Errorf("No product factory queue created")
	}

	c.factory.queueLength = uint64(len(c.factory.queue))
	c.factory.initialized = true

	return c.factory.nProducts, nil
}

// ProductFactoryNext is an iterator that can deliver batches of products
// after NewProductFactory was called
func (c *Connection) ProductFactoryNext() (products []Product, done bool, err error) {
	if c.factory.it >= c.factory.queueLength-1 {
		return products, true, nil
	}

	data, err := c.factory.queue[c.factory.it].execute()
	if err != nil {
		return products, false, err
	}

	for i := range data {
		response := new(Feed)
		err := json.Unmarshal(data[i], response)
		if err != nil {
			fmt.Println(err)
			break
		}
		for j := range response.Products {
			products = append(products, response.Products[j])
		}
	}

	if len(products) == 0 {
		return products, false, fmt.Errorf("No products retrieved from factory")
	}

	c.factory.it++

	return products, false, nil
}

// QueryCategories returns all the Categories in the active programs and the respective count of products
// LanguageCode: ISO 639-1 code of the language to use in the response. For example "en" for English or "sv" for Swedish.
func (c *Connection) QueryCategories(languageCode string) (ProductCategories, error) {
	var categoryTree ProductCategories

	// ignores language params that are not 2 letter codes
	queryString := ""
	if len(languageCode) == 2 {
		queryString = ";language=" + strings.ToLower(languageCode)
	}

	var r = getRequest{
		Connection: c,
		Endpoint:   "productCategories" + queryString,
	}

	data, err := r.Send()
	if err != nil {
		return categoryTree, err
	}

	err = json.Unmarshal(data, &categoryTree)
	if err != nil {
		return categoryTree, err
	}

	return categoryTree, nil
}

// QueryFeeds returns all the active Feeds from the Product Feed Sevice
// http://api.tradedoubler.com/1.0/productFeeds{/feedId}[.xml|.json|empty]?token={token}[&jsonp=myCallback]
func (c *Connection) QueryFeeds(languageCode string) (feeds []FeedInfo, err error) {
	var response map[string][]FeedInfo

	if len(languageCode) != 2 {
		return feeds, fmt.Errorf("Not a proper ISO language code - %s", languageCode)
	}

	var r = getRequest{
		Connection: c,
		Endpoint:   "productFeeds",
	}

	data, err := r.Send()
	if err != nil {
		return feeds, fmt.Errorf("Failed to query feed info - %s", string(data))
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return feeds, err
	}

	if len(response["feeds"]) == 0 {
		return feeds, fmt.Errorf("No feeds returned")
	}

	for i := range response["feeds"] {
		if !response["feeds"][i].Active && !response["feeds"][i].Secret {
			continue
		}
		if response["feeds"][i].LanguageISOCode == languageCode {
			feeds = append(feeds, response["feeds"][i])
		}
	}

	return feeds, nil
}
