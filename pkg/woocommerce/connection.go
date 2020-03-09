package woocommerce

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"stillgrove.com/gofeedyourself/pkg/cache"
	c "stillgrove.com/gofeedyourself/pkg/collection"
	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	gwc "stillgrove.com/gofeedyourself/pkg/woocommerce/client"
)

const (
	// MaxRetries - retries for requests
	MaxRetries = 1
	// BatchStrideSize - number of products per request batch - max 100
	BatchStrideSize = 64
	// MaxConcurrentRequests - number of concurrent requests
	MaxConcurrentRequests = 4
	// AllowMultiCats - Products can have multiple category IDs (disabled for testing)
	AllowMultiCats = false
)

type wooCredentials struct {
	domain string
	key    string
	secret string
}

// WooConnection interfaces with the WooCommerce backend
type WooConnection struct {
	initialized  bool
	credentials  wooCredentials
	Connection   *gwc.Client
	attributeMap map[string]int32
	brandMap     map[uint64]int32
	categoryMap  *map[string]map[string]int32
	//mappings     ProductMapping
	Locale string
}

// NewWooConnection takes in the credentials and initializes a WooConnection object
func NewWooConnection(domain, key, secret, locale string) (w WooConnection, err error) {
	if err != nil {
		return w, err
	}

	w.Locale = locale

	w.Connection, err = gwc.NewClient(
		domain,
		key,
		secret,
		"v3",
		true,
		BatchStrideSize,
		MaxConcurrentRequests,
		MaxRetries,
	)
	if err != nil {
		return w, err
	}

	w.initialized = true

	return w, nil
}

/* ------------------------------------------------------
-- Update Process  --------------------------------------
-------------------------------------------------------*/

// ApplyUpdate applies operations from request queue of "name" and does one of three things:
// - output = "api" : Uploads To Woocommerce
// - output = "json" : Writes to a JSON file
// - output = "csv" : Writes to a CSV file
func (w *WooConnection) ApplyUpdate(name string, output string) (err error) {
	switch output {
	case "json", "csv", "api":
	default:
		return fmt.Errorf("Unknown output parameter - %s", output)
	}
	// if in debug mode, output update requests to file and return
	if output != "api" {
		log.Infof("Not in production mode, skipping update")

		fname := fmt.Sprintf(helpers.FindFolderDir("gofeedyourself")+"/logs/upload_%s.json", time.Now().Format("2006-01-02"))
		err := w.SaveUpdateToFile(fname, output)
		if err != nil {
			return fmt.Errorf("Failed to save update to file - %v", err)
		}
		return nil
	}

	log.WithField("Request Queue", name).Infoln("Executing Queue")
	_, err = w.Connection.ExecuteRequestQueue(name, false, true)
	if err != nil {
		fname := fmt.Sprintf(helpers.FindFolderDir("gofeedyourself")+"/logs/failed_request_%s.json", time.Now().Format("2006-01-02"))
		err2 := w.SaveUpdateToFile(fname, "json")
		if err2 != nil {
			return fmt.Errorf("%v - %v", err, err2)
		}
		return err
	}

	return nil
}

// SaveUpdateToFile dumps the content of the request queue to a file so the requests can be analyzed
func (w *WooConnection) SaveUpdateToFile(filename string, filetype string) error {
	switch filetype {
	case "csv", "json":
	default:
		return fmt.Errorf("Unknown filetype - %s", filetype)
	}
	requests, err := w.Connection.ViewRequestQueue()
	if err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf("%s.%s", filename, filetype))
	if err != nil {
		return err
	}
	defer f.Close()

	for i := range requests {
		_, err = f.Write(requests[i])
		if err != nil {
			return err
		}
	}
	return nil
}

/*------------------------------------------------------
-- Request Queues --------------------------------------
-------------------------------------------------------*/

// PrepareUpdate merges existing products in the WooCommerce backend with the suggested update
func (w *WooConnection) PrepareUpdate(products *feed.ProductMap, categoryMap map[string]map[string][]*int32, productionFlag, purgeFlag bool) (err error) {
	if w.initialized == false {
		err = errors.New("Please initialize with your credentials first. WooConnection.Init()")
		return fmt.Errorf("Update products in WC backend - %v", err)
	}

	var outProducts uint64

	inProducts, inFeeds, inCategories := products.Stats()
	if inProducts == 0 {
		return fmt.Errorf("No products in map")
	}
	log.Printf("Preparing %d Products from %d feeds with %d categories\n", inProducts, inFeeds, inCategories)

	mappings, err := w.PrepareMappings(products, categoryMap, productionFlag)
	if err != nil {
		return fmt.Errorf("Prepare Updates - %v", err)
	}

	newProducts, err := PMFromPM(products, &mappings)
	if err != nil {
		return fmt.Errorf("Convert products to WC - %v", err)
	}

	oldProductMap, err := w.GetOldProductMap(productionFlag)
	if err != nil {
		return fmt.Errorf("Fetch current products from the WC backend - %v", err)
	}

	create, update, delete, err := newProducts.GetGroups(oldProductMap)
	if err != nil {
		return fmt.Errorf("Prepare create/update/delete groups - %v", err)
	}
	newProducts.Flush()

	if productionFlag == true {
		err = w.BuildDeleteProductQueue(delete)
		if err != nil {
			return fmt.Errorf("Build product delete queue- %v", err)
		}
	}
	err = w.BuildCreateUpdateProductQueue(create, update, purgeFlag)
	if err != nil {
		return fmt.Errorf("Build product update queue- %v", err)
	}
	outProducts = uint64(len(create) + len(update))

	log.Printf("The following changes will be made: Create %d, Update %d, Delete %d\n", len(create), len(update), len(delete))
	log.Printf("%d Products from feeds, %d in update, \n", inProducts, outProducts)

	err = w.ValidateUpdate()
	if err != nil {
		return fmt.Errorf("Request Validation Failed - %v", err)
	}

	return nil
}

//ValidateUpdate checks the raw requests before they are being sent out
func (w *WooConnection) ValidateUpdate() error {
	req, err := w.Connection.ViewRequestQueue()
	if err != nil {
		return fmt.Errorf("View update requests - %v", err)
	}

	var exist bool
	var essentials = [...]string{
		"sku",
		"type",
		"button_text",
		"stock_status",
		"description",
	}

	updates := struct {
		Lang   string                   `json:"lang"`
		Update []map[string]interface{} `json:"update"`
		Create []map[string]interface{} `json:"create,omitempty"`
	}{}
	for i := range req {
		err = json.Unmarshal(req[i], &updates)
		if err != nil {
			return fmt.Errorf("Validate update requests - %v", err)
		}

		for j := range updates.Update {
			_, exist = updates.Update[j]["id"]
			if !exist {
				return fmt.Errorf("Missing id in update request")
			}
			for k := range essentials {
				_, exist = updates.Update[j][essentials[k]]
				if !exist {
					return fmt.Errorf("Missing field in Update: %s", essentials[k])
				}
			}
		}
		for j := range updates.Create {
			_, exist = updates.Create[j]["id"]
			if exist {
				updates.Create[j]["id"] = 0
				return fmt.Errorf("Can't create product with ID preset")
			}
			_, exist = updates.Create[j]["images"]
			if exist {
				img := updates.Create[j]["images"].([]interface{})
				for i := range img {
					_, exist = img[i].(map[string]interface{})["id"]
					if exist {
						return fmt.Errorf("Upload can't have image ID - %v", updates.Create[j]["images"].([]map[string]interface{})[i])
					}
				}
			}

			for k := range essentials {
				_, exist = updates.Create[j][essentials[k]]
				if !exist {
					return fmt.Errorf("Missing field in Create: %s", essentials[k])
				}
			}
		}
	}
	return nil
}

func (w *WooConnection) BuildDeleteProductQueue(delete []int) (err error) {
	if w.initialized == false {
		return fmt.Errorf("Please initialize with your credentials first. WooConnection.Init()")
	}
	vars := struct {
		endpoint string
		np       uint64
		i        uint64
		idx      uint64
		exist    bool
	}{
		endpoint: "products/batch",
	}

	r := gwc.BatchPostRequest{
		Endpoint: vars.endpoint,
		Locale:   w.Locale,
	}

	nd := uint64(len(delete))
	for i := range delete {
		r.Delete = append(r.Delete, delete[i])

		vars.i++
		if vars.i%uint64(BatchStrideSize) == 0 || vars.i >= nd-1 {
			w.Connection.PushToQueue("delete", r)
			r = gwc.BatchPostRequest{
				Endpoint: vars.endpoint,
				Locale:   w.Locale,
			}
		}
	}

	return nil
}

func (w *WooConnection) BuildCreateUpdateProductQueue(create map[uint64]*Product, update map[uint64]*Product, purgeFlag bool) (err error) {
	vars := struct {
		err                error
		np                 uint64
		i                  uint64
		exist              bool
		endpoint           string
		nupdates, ncreates uint64
	}{
		endpoint: "products/batch",
	}

	r := gwc.BatchPostRequest{
		Endpoint: vars.endpoint,
		Locale:   w.Locale,
	}

	if w.initialized == false {
		return fmt.Errorf("Please initialize with your credentials first. WooConnection.Init()")
	}
	if len(w.attributeMap) < 0 {
		return fmt.Errorf("Please create the attribute map first so attributes can be mapped properly")
	}

	nc := uint64(len(create))
	for k := range create {
		if create[k].Name == "" {
			continue
		}
		r.Create = append(r.Create, *create[k])

		vars.i++
		if vars.i%uint64(BatchStrideSize) == 0 || vars.i >= nc-1 {
			w.Connection.PushToQueue("createupdate", r)
			r = gwc.BatchPostRequest{
				Endpoint: vars.endpoint,
				Locale:   w.Locale,
			}
		}
	}

	nu := uint64(len(update))
	for k := range update {
		if update[k].GetID() == 0 {
			continue
		}
		if !purgeFlag {
			upd := *update[k]
			upd.Images = nil

			r.Update = append(r.Update, *update[k])
		} else {
			r.Update = append(r.Update, *update[k])
		}

		vars.i++
		if vars.i%uint64(BatchStrideSize) == 0 || vars.i >= nu-1 {
			w.Connection.PushToQueue("createupdate", r)
			r = gwc.BatchPostRequest{
				Endpoint: vars.endpoint,
				Locale:   w.Locale,
			}
		}
	}

	return nil
}

// PrepareMappings returns mappings object to be used for product conversion
func (w *WooConnection) PrepareMappings(newProductMap *feed.ProductMap, categoryMap map[string]map[string][]*int32, productionFlag bool) (mappings ProductMapping, err error) {
	mappings.categoryMap = categoryMap
	mappings.brandMap, err = w.generateBrandMap(newProductMap)
	if err != nil {
		return mappings, fmt.Errorf("Synchronize Brands - %v", err)
	}
	mappings.attributeMap, err = w.prepareAttributes(newProductMap, productionFlag)
	if err != nil {
		return mappings, fmt.Errorf("Check/update attributes in WC backend - %v", err)
	}
	mappings.discountBinSize = 10

	return mappings, nil
}

// prepareAttributes loads registered attributes and creates new ones if need be
func (w *WooConnection) prepareAttributes(newProductMap *feed.ProductMap, applyUpdate bool) (attributeMap map[string]*int32, err error) {
	if w.initialized == false {
		return attributeMap, fmt.Errorf("Please initialize with your credentials first. WooConnection.Init()")
	}

	newAttributeMap, err := extractAttributeMap(newProductMap)
	if err != nil {
		return attributeMap, fmt.Errorf("Extracting attributes from new product feed - %v", err)
	}

	currentAttributeMap, err := w.fetchCurrentAttributeMap()
	if err != nil {
		return attributeMap, fmt.Errorf("Loading current attributes - %v", err)
	}

	var createAttributesReq = gwc.BatchPostRequest{
		Endpoint: "products/attributes/batch",
		Locale:   w.Locale,
	}

	var hasUpdates bool
	for k := range newAttributeMap {
		_, exists := currentAttributeMap[k]
		if exists == false {
			hasUpdates = true
			createAttributesReq.Create = append(
				createAttributesReq.Create,
				gwc.Attribute{
					Name:   k,
					Type:   "select",
					Locale: w.Locale,
				},
			)
		}

		delete(newAttributeMap, k)
	}

	// execute synchronously
	if hasUpdates == true && applyUpdate == true {
		_, err := createAttributesReq.Send(w.Connection)
		if err != nil {
			return attributeMap, fmt.Errorf("Failed to create attributes - %v", err)
		}
		//Update oldAttributes
		currentAttributeMap, err = w.fetchCurrentAttributeMap()
		if err != nil {
			return attributeMap, fmt.Errorf("Updating attributes - %v", err)
		}
	}

	attributeMap = make(map[string]*int32, len(currentAttributeMap))
	for key := range currentAttributeMap {
		if currentAttributeMap[key].ID == 0 {
			return attributeMap, fmt.Errorf("Attribute returned without ID - %v", key)
		}
		attributeMap[key] = &currentAttributeMap[key].ID
	}

	return attributeMap, nil
}

func extractAttributeMap(productMap *feed.ProductMap) (map[string]map[string]struct{}, error) {
	var attributeMap = map[string]map[string]struct{}{
		"Brand":          make(map[string]struct{}),
		"Color":          make(map[string]struct{}),
		"Size":           make(map[string]struct{}),
		"Store":          make(map[string]struct{}),
		"Pattern":        make(map[string]struct{}),
		"Gender":         make(map[string]struct{}),
		"Color Group":    make(map[string]struct{}),
		"Discount Level": make(map[string]struct{}),
	}

	pm, _, _, _ := productMap.Get()
	for _, v := range pm {
		_, exist := attributeMap["Brand"][v.Brand]
		if exist == false {
			attributeMap["Brand"][v.Brand] = struct{}{}
		}

		_, exist = attributeMap["Color"][v.Color]
		if exist == false {
			attributeMap["Color"][v.Color] = struct{}{}
		}

		_, exist = attributeMap["Gender"][v.Gender]
		if exist == false {
			attributeMap["Gender"][v.Gender] = struct{}{}
		}

		for _, bin := range v.DiscountBins {
			_, exist = attributeMap["Discount Level"][bin]
			if exist == false {
				attributeMap["Discount Level"][bin] = struct{}{}
			}
		}

		for _, group := range v.ColorGroups {
			_, exist = attributeMap["Color Group"][group]
			if exist == false {
				attributeMap["Color Group"][group] = struct{}{}
			}
		}

		for _, pattern := range v.Patterns {
			_, exist = attributeMap["Pattern"][pattern]
			if exist == false {
				attributeMap["Pattern"][pattern] = struct{}{}
			}
		}

		for i := range v.Retailers {
			_, exist = attributeMap["Store"][v.Retailers[i].Name]
			if exist == false {
				attributeMap["Store"][v.Retailers[i].Name] = struct{}{}
			}
			for _, size := range v.Retailers[i].Sizes {
				_, exist = attributeMap["Size"][size]
				if exist == false {
					attributeMap["Size"][size] = struct{}{}
				}
			}
		}
	}

	return attributeMap, nil
}

func (w *WooConnection) fetchCurrentAttributeMap() (currentAttributeMap map[string]*gwc.Attribute, err error) {
	var r = gwc.GetRequest{
		Endpoint: "products/attributes",
		Params: url.Values{
			"lang": []string{
				w.Locale,
			},
		},
	}

	raw, err := r.Send(w.Connection)
	if err != nil {
		return currentAttributeMap, err
	}

	var attributes []gwc.Attribute
	err = json.Unmarshal(raw, &attributes)
	if err != nil {
		return currentAttributeMap, err
	}

	currentAttributeMap = make(map[string]*gwc.Attribute)

	for i := range attributes {
		if attributes[i].ID == 0 {
			return currentAttributeMap, fmt.Errorf("Missing attribute - %v", attributes[i])
		}
		currentAttributeMap[attributes[i].Name] = &attributes[i]
	}

	return currentAttributeMap, nil
}

// GetBrandMap generates a name <-> id map for PerfectWooCommerce Brands
func (w *WooConnection) generateBrandMap(newProductMap *feed.ProductMap) (brandMap map[uint64]*int32, err error) {
	brandMap, err = w.fetchBrandMap()
	if err != nil {
		return brandMap, fmt.Errorf("Fetch existing brand map - %v", err)
	}

	newBrands := newProductMap.GetBrands()

	var hasUpdates bool
	for _, brand := range newBrands {
		_, exists := brandMap[c.HashKey(brand)]
		if exists == true {
			continue
		}
		w.Connection.PushToQueue(
			"brandmap",
			gwc.PostRequest{
				Endpoint: "brands",
				Payload: gwc.Brand{
					Name:   brand,
					Locale: w.Locale,
				},
				Locale: w.Locale,
			},
		)
		hasUpdates = true
	}
	newBrands = nil

	if hasUpdates == true {
		_, err = w.Connection.ExecuteRequestQueue("brandmap", false, true)
		if err != nil {
			return brandMap, fmt.Errorf("Creating new brands map - %v", err)
		}
		// back off a little bit not to run into capacity issues
		time.Sleep(2 * time.Second)
		brandMap, err = w.fetchBrandMap()
		if err != nil {
			return brandMap, fmt.Errorf("Loading updated brand map - %v", err)
		}
	}

	return brandMap, nil
}

func (w *WooConnection) fetchBrandMap() (brandMap map[uint64]*int32, err error) {
	var r = gwc.GetRequest{
		Endpoint: "brands",
		Params: url.Values{
			"lang": []string{
				w.Locale,
			},
		},
	}
	raw, err := r.Send(w.Connection)
	if err != nil {
		return brandMap, fmt.Errorf("Loading brand map - send request - %v", err)
	}

	var brands []gwc.Brand
	err = json.Unmarshal(raw, &brands)
	if err != nil {
		return brandMap, fmt.Errorf("Unmarshal brand map - %v", err)
	}

	var key uint64
	brandMap = make(map[uint64]*int32, len(brands))
	for i := range brands {
		key = c.HashKey(brands[i].Name)
		brandMap[key] = &brands[i].TermID
	}

	return brandMap, nil
}

func (w *WooConnection) GetOldProductMap(productionFlag bool) (productMap map[uint64]uint64, err error) {
	file := helpers.FindFolderDir("gofeedyourself") + "/cache/" + time.Now().Format("2006-01-02") + "_wc"
	cache, err := cache.NewBadgerCache(file, 4*time.Hour)
	if err != nil {
		return productMap, fmt.Errorf("Init cache - %v", err)
	}
	defer cache.Close()

	loadPageSize := 100
	endpoint := "products"

	totalNumProducts, err := w.Connection.GetNumItems(endpoint, w.Locale) //get total number of items from product endpoint
	if err != nil {
		return productMap, fmt.Errorf("Get number of items - %v", err)
	}

	if !productionFlag {
		if totalNumProducts > 500 {
			totalNumProducts = 500
		}
	}

	if totalNumProducts == 0 {
		log.Println("0 Products currently in the database")
		return productMap, nil
	}

	if loadPageSize > totalNumProducts {
		loadPageSize = totalNumProducts
	}

	for offset := 0; offset < totalNumProducts; offset += loadPageSize {
		w.Connection.PushToQueue(
			"oldProducts",
			gwc.GetRequest{
				Endpoint: endpoint,
				Params: url.Values{
					"offset": []string{
						strconv.Itoa(offset),
					},
					"per_page": []string{
						strconv.Itoa(loadPageSize),
					},
				},
			},
		)
	}
	rawResponse, err := w.Connection.ExecuteRequestQueue("oldProducts", true, false)
	if err != nil {
		return productMap, fmt.Errorf("Get old products - %v", err)
	}

	for i := range rawResponse {
		if len(rawResponse[i]) < 1 {
			continue
		}
		err = cache.Store(
			map[string][]byte{
				fmt.Sprintf("%d", i): rawResponse[i],
			},
		)
		if err != nil {
			return productMap, fmt.Errorf("Cache Old Products - %v", err)
		}
	}

	productMap = make(map[uint64]uint64, totalNumProducts)
	stored, err := cache.LoadAll()
	if err != nil {
		return productMap, fmt.Errorf("Retrieve Old Products From Cache - %v", err)
	}

	var key int32
	for _, v := range stored {
		var oldProducts []gwc.Product

		err := json.Unmarshal(v, &oldProducts)
		if err != nil {
			return productMap, fmt.Errorf("Retrieve Old Products From Cache - %v", err)
		}

		for i := range oldProducts {
			key = oldProducts[i].GetKey()
			productMap[uint64(key)] = oldProducts[i].ID
		}
	}

	return productMap, nil
}
