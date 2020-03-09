// +build unit
// +build !integration

package tradedoubler

import (
	"testing"

	c "stillgrove.com/gofeedyourself/pkg/collection"
	f "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	gtd "stillgrove.com/gofeedyourself/pkg/tradedoubler/client"
)

func getTestProduct() Product {
	var terms = [...]string{
		"green",
		"5xl",
		"men",
		"women",
		"melange",
		"hotpants",
		"jeans",
		"shorts",
		"short-shorts",
	}
	return Product{
		gtd.Product{
			Name:        "test1 hotpants",
			Description: "testtestest",
			Brand:       "testbrand",
			Language:    "sv_se",
			Categories: []gtd.Category{
				gtd.Category{Name: "women > short-shorts"},
			},
			ProductImage: gtd.Image{
				URL: "www.test.de",
			},
			Identifiers: map[string]string{
				"SKU": "abc123",
			},
			Fields: []map[string]interface{}{
				map[string]interface{}{
					"name":  "Color",
					"value": "Olde Slimeball",
				},
				map[string]interface{}{
					"name":  "Sizes",
					"value": "s,M,l,XXxXXl",
				},
				map[string]interface{}{
					"name":  "gender",
					"value": "WOMEN",
				},
				map[string]interface{}{
					"name":  "subcategorypath",
					"value": "/jeans/shorts",
				},
			},
			Offers: []gtd.Offer{
				gtd.Offer{
					ProductURL:   "www.test.de",
					Availability: "instock",
					PriceHistory: []gtd.Price{
						gtd.Price{
							Date: 10000,
							Price: map[string]string{
								"value":    "2690",
								"currency": "SEK",
							},
						},
					},
				},
			},
		}, &feed.Mapping{
			ColorMap: map[string][]*string{
				"slimeball": []*string{
					&terms[0],
				},
			},
			SizeMap: map[string][]*string{
				"xxxxl": []*string{
					&terms[1],
				},
			},
			GenderMap: map[string][]*string{
				"men": []*string{
					&terms[2],
				},
				"women": []*string{
					&terms[3],
				},
			},
			PatternMap: map[string][]*string{
				"slime": []*string{
					&terms[4],
				},
			},
			CatNameMap: map[string][]*string{
				"hotpants":     []*string{&terms[5]},
				"jeans":        []*string{&terms[6]},
				"shorts":       []*string{&terms[7]},
				"short-shorts": []*string{&terms[8]},
			},
			ConversionMap: map[int32]*f.Product{},
		},
	}
}

func TestProducts(t *testing.T) {
	tdp := getTestProduct()
	fp, err := tdp.ToFeedProduct()
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = fp.Validate()
	if err != nil {
		t.Fatalf("%v", err)
	}

	var checklist = []string{
		"hotpants",
		"jeans",
		"shorts",
	}
	var counter uint8
	for i := range fp.ProviderCategories {
		if c.StringInList(fp.ProviderCategories[i].Name, checklist) {
			counter++
		}
	}
	if counter != 3 {
		t.Fatalf("Failed to extract categories - %v", fp.ProviderCategories)
	}

	if len(fp.ColorGroups) == 0 {
		t.Fatalf("Failed to map color - %v", fp.Color)
	}

	if len(fp.Retailers) == 0 || len(fp.Retailers) != len(fp.RetailerMap) {
		t.Fatalf("Retailers scrambled - %v - %v", fp.Retailers, fp.RetailerMap)
	}

	if len(fp.Retailers[0].Sizes) < 2 {
		t.Fatalf("Sizes not extracted correctly - %v", fp.Retailers[0].Sizes)
	}
}
