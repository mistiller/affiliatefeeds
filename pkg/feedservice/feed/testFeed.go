package feed

// TestFeed is a fully working Feed object for testing
type TestFeed struct {
	Name string
}

// NewTestFeed returns a realistic feed object for testing
func NewTestFeed(name string) TestFeed {
	return TestFeed{
		Name: name,
	}
}

// GetName returns the test feed's name
func (t TestFeed) GetName() string {
	return t.Name
}

func (c TestFeed) GetLocale() *Locale {
	l, err := NewLocale("SE", "sv", "sv_se")
	if err != nil {
		return &Locale{}
	}
	return l
}

// Get returns an array of feed products for testing
func (t TestFeed) Get(productionFlag bool) ([]Product, error) {
	return []Product{
		Product{
			Name:             "Testproduct1",
			Description:      "This is a test product",
			ShortDescription: "Really: A test product",
			SKU:              "ABC123",
			ImageURL:         "www.images.com",
			Gender:           "Female",
			Brand:            "Testbrand1",
			ColorGroups: []string{
				"Green",
			},
			Color: "Slimeball",
			Patterns: []string{
				"Checkers",
			},
			HighestPrice: 129.99,
			FromFeeds:    []int32{111},
			RetailerMap: map[uint64]struct{}{
				56789: struct{}{},
			},
			Retailers: []Retailer{
				Retailer{
					Name: "The Store",
					Link: "www.store.com/testproduct1",
					Sizes: []string{
						"XXL",
					},
					Price:        "129.99",
					Currency:     "SEK",
					Availability: "instock",
				},
			},
			ProviderCategories: []ProviderCategory{
				ProviderCategory{
					ProviderName: t.GetName(),
					Name:         "Pants",
					Gender:       'w',
				},
				ProviderCategory{
					ProviderName: t.GetName(),
					Name:         "Jeans",
					Gender:       'w',
				},
			},
			Language: "sv_se",
		},
		Product{
			Name:     "Testproduct1",
			SKU:      "ABC123",
			ImageURL: "www.images.com",
			Gender:   "Female",
			Brand:    "Testbrand1",
			ColorGroups: []string{
				"Green",
			},
			Color: "Slimeball",
			Patterns: []string{
				"Checkers",
			},
			HighestPrice: 129.99,
			FromFeeds:    []int32{99},
			RetailerMap: map[uint64]struct{}{
				1234567: struct{}{},
			},
			Retailers: []Retailer{
				Retailer{
					Name: "The Shack",
					Link: "www.shack.com/testproduct1",
					Sizes: []string{
						"XL",
					},
					Price:        "139.99",
					Currency:     "SEK",
					Availability: "instock",
				},
			},
			ProviderCategories: []ProviderCategory{
				ProviderCategory{
					ProviderName: t.GetName(),
					Name:         "Jeans",
					Gender:       'w',
				},
				ProviderCategory{
					ProviderName: t.GetName(),
					Name:         "Women",
					Gender:       'w',
				},
			},
			Language:        "sv_se",
			WebsiteFeatures: 10,
		},
		Product{
			Name:             "Testproduct2",
			Description:      "This is the other test product",
			ShortDescription: "Really, really: A test product",
			SKU:              "DEF456",
			ImageURL:         "www.images.com",
			Gender:           "Male",
			Brand:            "Testbrand2",
			ColorGroups:      []string{},
			Color:            "Umbra",
			Patterns:         []string{},
			FromFeeds:        []int32{99},
			HighestPrice:     199.99,
			LowestPrice:      129.99,
			ProviderCategories: []ProviderCategory{
				ProviderCategory{
					ProviderName: t.GetName(),
					Name:         "Shirts",
					Gender:       'm',
				},
			},
			RetailerMap: map[uint64]struct{}{
				998877: struct{}{},
				998878: struct{}{},
				998879: struct{}{},
			},
			Retailers: []Retailer{
				Retailer{
					Name: "The Shack",
					Link: "www.shack.com/testproduct2",
					Sizes: []string{
						"XL",
					},
					Price:        "199.99",
					Currency:     "SEK",
					Availability: "out of stock",
				},
				Retailer{
					Name: "The Store",
					Link: "www.store.com/testproduct2",
					Sizes: []string{
						"L",
					},
					Price:        "189.99",
					Currency:     "SEK",
					Availability: "instock",
				},
				Retailer{
					Name: "The Place",
					Link: "www.place.com/testproduct2",
					Sizes: []string{
						"L",
					},
					Price:        "129.99",
					Currency:     "SEK",
					Availability: "instock",
				},
			},
			Language: "sv_se",
		},
	}, nil
}
