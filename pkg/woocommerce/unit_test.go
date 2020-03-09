// +build unit
// +build !integration

package woocommerce

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	gwc "stillgrove.com/gofeedyourself/pkg/woocommerce/client"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

func getExamples() (pMap *feed.ProductMap, dummyCategories map[string]map[string][]*int32, err error) {
	f := feed.NewTestFeed("TestProducts")
	products, err := f.Get(false)
	if err != nil {
		return pMap, dummyCategories, fmt.Errorf("Load test feed - %v", err)
	}

	pMap, err = feed.PMFromSlice(products)
	if err != nil {
		return pMap, dummyCategories, fmt.Errorf("Convert to product map - %v", err)
	}

	var p, j, s int32 = 997, 998, 999
	dummyCategories = map[string]map[string][]*int32{
		"w": map[string][]*int32{
			"pants":  []*int32{&p},
			"jeans":  []*int32{&j, &p},
			"shirts": []*int32{&s},
		},
		"m": map[string][]*int32{
			"shirts": []*int32{&s},
		},
	}

	return pMap, dummyCategories, nil
}

func getConnection() (c WooConnection, err error) {
	c, err = NewWooConnection(
		"https://www.stillgrove.com",
		os.Getenv("WOO_KEY"),
		os.Getenv("WOO_SECRET"),
		"sv_se",
	)
	if err != nil {
		return c, err
	}
	return c, nil
}

func TestConnectionUnit(t *testing.T) {
	if !helpers.IsOnline("") {
		panic("Currently offline")
	}

	c, err := getConnection()
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.fetchCurrentAttributeMap()
	if err != nil {
		t.Fatalf("%v", err)
	}
	_, err = c.fetchBrandMap()
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, err = c.GetOldProductMap(false)
	if err != nil {
		t.Fatalf("Fetch current products from the WC backend - %v", err)
	}
}

func TestCRUD(t *testing.T) {
	var err error
	c, err := getConnection()
	if err != nil {
		t.Fatal(err)
	}

	var create = []byte(`
		{
			"sku":"3932526995",
			"name":"Tjw Tommy Badge Tee T-shirts \u0026 Tops Short-sleeved Svart Tommy Jeans",
			"type":"external",
			"description":"Tommy Jeans Tjw Tommy Badge Tee",
			"short_description":"Tommy Jeans Tjw Tommy Badge Tee",
			"regular_price":"500.00",
			"external_url":"https://pdt.tradedoubler.com/click?a(3072116)p(227648)product(40726677-95eb-4ae0-8305-ec8c3dad4369)ttid(3)url(https%3A%2F%2Fwww.boozt.com%2Fse%2Fsv%2Ftommy-jeans%2Ftjw-tommy-badge-tee-_20248309%2F20248341)",
			"button_text":"Boozt.com",
			"stock_status":"instock",
			"dimensions":{},
			"categories":[{
					"id":15186,
					"name":"",
					"image":{},
					"_links":{}
				},{
					"id":2859,
					"name":"",
					"image":{},
					"_links":{}
				},{
					"id":2877,
					"name":"",
					"image":{},
					"_links":{}
				}
			],
			"images":[{
				"src":"https://ean-images.booztcdn.com/tommy-jeans/1300x1700/tjsdw0dw06813_ctommyblack_v078.jpg",
				"name":"Tjw Tommy Badge Tee T-shirts \u0026 Tops Short-sleeved Svart Tommy Jeans-tommy black",
				"alt":"Tjw Tommy Badge Tee T-shirts \u0026 Tops Short-sleeved Svart Tommy Jeans-tommy black"
			}],
			"attributes":[{
				"id":4,
				"name":"Size",
				"option":"S",
				"options":["S","XL","M","XS"],
				"visible":true,
				"lang":"sv_se"
			},{
				"id":3,
				"name":"Color",
				"option":"black",
				"options":["black"],
				"visible":true,
				"lang":"sv_se"
			},{
				"id":7,
				"name":"Gender",
				"option":"women",
				"options":["women"],
				"visible":true
			},{
				"id":5,
				"name":"Store",
				"option":"Boozt.com",
				"options":["Boozt.com"],
				"visible":true
			},{
				"id":9,
				"name":"Brand",
				"option":"Tommy Jeans",
				"options":["Tommy Jeans"],
				"visible":true
			}],
			"brands":[1275],
			"language":"sv_se",
			"lang":"sv_se",
			"custom_prices":{
				"SEK":{
					"regular_price":"500.00"
				}
			}
		}
	`)

	var prod gwc.Product
	err = json.Unmarshal(create, &prod)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(prod)

	var req = gwc.PostRequest{
		Endpoint: "products/batch",
		Locale:   "sv_se",
		Payload:  prod,
	}
	resp, err := req.Send(c.Connection)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(string(resp))
}

func TestCategoriesUnit(t *testing.T) {
	var (
		err error
	)
	pm, dummyCategories, err := getExamples()
	if err != nil {
		t.Fatalf("Get examples - %v", err)
	}

	products, _, _, _ := pm.Get()
	var cats []int32
	temp := new(FeedProduct)
	for k := range products {
		temp = &FeedProduct{
			*products[k],
		}

		// Allow MultiCats
		cats, err = GetWCCategories(temp.ProviderCategories, dummyCategories, true)
		if err != nil {
			t.Fatalf("%v", err)
		}
		fmt.Println(cats)
		for i := range cats {
			if cats[i] == 0 {
				t.Fatal("Returned Zero Category ID")
			}
		}

		// Force Single Categories
		cats, err = GetWCCategories(temp.ProviderCategories, dummyCategories, false)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(cats)
		if len(cats) != 1 {
			t.Fatalf("Failed to constrain categories to one")
		}
	}
}

func TestHelpers(t *testing.T) {
	var attributes = [...]string{
		"Brand",
		"Color",
		"Size",
		"Store",
		"Pattern",
		"Gender",
		"Color Group",
	}

	products, _, err := getExamples()
	if err != nil {
		t.Fatalf("%v", err)
	}
	mp, err := extractAttributeMap(products)
	if err != nil {
		t.Fatalf("Failed to extract map! - %v", err)
	}

	var exists bool
	for _, key := range attributes {
		_, exists = mp[key]
		if exists == false {
			t.Fatalf("Extract attributes failed: %s", key)
		}
	}
}

func TestGroups(t *testing.T) {
	var np = map[uint64]*Product{
		1: &Product{
			gwc.Product{Name: "abc"},
			"",
			0.0,
			0.0,
			[]string{},
		},
		2: &Product{
			gwc.Product{Name: "def"},
			"",
			0.0,
			0.0,
			[]string{},
		},
		3: &Product{
			gwc.Product{Name: "ghi"},
			"",
			0.0,
			0.0,
			[]string{},
		},
	}

	var op = map[uint64]uint64{
		99: 100,
		1:  111,
	}
	c, u, d, err := getGroups(np, op)
	if err != nil {
		t.Fatal(err)
	}

	if len(c) != 2 ||
		len(u) != 1 ||
		len(d) != 1 {
		t.Fatalf("Incorrect grouping - create: %v, update: %v, delete %v", c, u, d)
	}
}
