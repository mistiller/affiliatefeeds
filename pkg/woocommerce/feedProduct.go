package woocommerce

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	c "stillgrove.com/gofeedyourself/pkg/collection"
	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	f "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	gwc "stillgrove.com/gofeedyourself/pkg/woocommerce/client"
)

var (
	// Locales lists whitelisted locale codes
	Locales = []string{
		"sv_se",
	}
)

// FeedProduct wraps around the Woocommerce Product to add specific validation
type FeedProduct struct {
	f.Product
}

// ProductMapping stores product mapping tables
type ProductMapping struct {
	attributeMap    map[string]*int32
	brandMap        map[uint64]*int32
	categoryMap     map[string]map[string][]*int32
	discountBinSize int
}

// ToWooProduct takes in a feed product and returns a woocommerce connection product to be uploaded
func (p *FeedProduct) ToWooProduct(mappings *ProductMapping) (wp *Product, err error) {
	wp = &Product{
		gwc.Product{
			SKU:              p.SKU, //fmt.Sprint(p.Key),
			Name:             p.Name,
			Description:      c.CollateString(p.Description, p.ShortDescription),
			ShortDescription: c.CollateString(p.ShortDescription, p.Description),
			Type:             "external",
			MenuOrder:        p.CalculateRanking(),
			//Language:         p.Language,
			//Lang:             "sv_se",
			CustomPrices: make(map[string]gwc.WpmlPrice),
		},
		"",
		0.0,
		0.0,
		[]string{},
	}

	if !c.StringInList(p.Language, Locales) {
		return wp, fmt.Errorf("Unknonw locale: %s", p.Language)
	}
	wp.Language = p.Language
	wp.Lang = p.Language

	categories, err := GetWCCategories(p.ProviderCategories, mappings.categoryMap, AllowMultiCats)
	if err != nil {
		return wp, err
	}
	err = wp.AddCategories(categories)
	if err != nil {
		return wp, err
	}

	brandID, exists := mappings.brandMap[c.HashKey(p.Brand)]
	if exists {
		wp.Brands = append(
			wp.Brands,
			*brandID,
		)
	}

	err = wp.AddImage(
		p.ImageURL,
		p.Name+"-"+c.SanitizeHard(p.Color),
		p.Name+"-"+c.SanitizeHard(p.Color),
	)
	if err != nil {
		return wp, fmt.Errorf("%v", err)
	}

	err = p.processStore(wp, mappings)
	if err != nil {
		return wp, fmt.Errorf("ToWooProduct - Failed to process retailers")
	}

	attributeList, err := processAttributes(p, wp, mappings)
	if err != nil {
		return wp, fmt.Errorf("ToWooProduct - Failed to process attributes")
	}

	err = wp.Validate(attributeList)
	if err != nil {
		return wp, err
	}

	return wp, nil
}

func (p *FeedProduct) processStore(wp *Product, mappings *ProductMapping) (err error) {
	s := struct {
		price       float64
		lowestPrice float64
		currency    string
		lowestStr   string
		hasStore    bool
	}{
		lowestPrice: float64(p.HighestPrice),
	}

	for idx := range p.Retailers {
		if p.Retailers[idx].Availability != "instock" || c.IsEmpty(&p.Retailers[idx].Link) {
			continue
		}

		s.currency = p.Retailers[idx].Currency

		s.price, err = strconv.ParseFloat(p.Retailers[idx].Price, 32)
		if err != nil {
			continue
		}

		if s.price >= float64(p.HighestPrice) {
			p.HighestPrice = float32(s.price)
		}

		// check for low price && availability, this way the lowest available price can differ from p.LowestPrice
		if s.price <= s.lowestPrice || idx == 0 {
			if p.Retailers[idx].IsCrawler {
				continue
			}

			wp.ButtonText = p.Retailers[idx].Name
			wp.ExternalURL = p.Retailers[idx].Link

			wp.ActiveStore = p.Retailers[idx].Name

			wp.StockStatus = c.SanitizeHard(p.Retailers[idx].Availability)

			for n := range p.Retailers[idx].Sizes {
				wp.Sizes = append(wp.Sizes, p.Retailers[idx].Sizes[n])
			}

			s.lowestPrice = s.price
			s.hasStore = true
		}
	}
	if s.hasStore == false {
		return fmt.Errorf("No active Store")
	}

	wp.Sizes = c.UniqueNames(wp.Sizes)

	s.lowestStr = strconv.FormatFloat(s.lowestPrice, 'f', 2, 32)
	if s.lowestPrice < float64(p.HighestPrice) {
		wp.RegularPrice = strconv.FormatFloat(float64(p.HighestPrice), 'f', 2, 64)
		wp.SalePrice = s.lowestStr
	} else {
		wp.RegularPrice = s.lowestStr
	}

	wp.CustomPrices[s.currency] = gwc.WpmlPrice{
		RegularPrice: wp.RegularPrice,
		SalePrice:    wp.SalePrice,
	}

	wp.LowestPrice = float32(s.lowestPrice)
	if p.HighestPrice > wp.LowestPrice {
		wp.HighestPrice = p.HighestPrice
	} else {
		wp.HighestPrice = wp.LowestPrice
	}

	err = p.CalculateDiscounts(10)
	if err != nil {
		return fmt.Errorf("Recalculate discounts - %v", err)
	}

	return nil
}

func processAttributes(in *FeedProduct, out *Product, mappings *ProductMapping) (attributeList []string, err error) {
	var exist bool
	var multiAttributes = map[string][]string{
		"Size": out.Sizes,
	}

	// Sice we have so many products, kick out all the one swithout proper color group
	// REMOVE THIS FIRST IF TOO FEW PRODUCTS!
	if len(in.ColorGroups) == 0 {
		return attributeList, fmt.Errorf("Color wasn't mapped - %s", in.Color)
	}
	//multiAttributes["Color"] = in.Color
	multiAttributes["Color Group"] = in.ColorGroups

	// disabling pattern attributes for now, quality of input too low
	if false { // len(in.Patterns) > 0 {
		multiAttributes["Pattern"] = in.Patterns
	}
	if len(in.DiscountBins) > 0 {
		multiAttributes["Discount Level"] = in.DiscountBins
	}

	for k := range multiAttributes {
		v := multiAttributes[k]
		if len(v) == 0 {
			continue
		}

		_, exist = mappings.attributeMap[k]
		if !exist {
			log.WithField("Name", k).Warningln("Not found in attribute map")
			continue
		}

		out.Attributes = append(
			out.Attributes,
			gwc.Attribute{
				Name:    k,
				ID:      *mappings.attributeMap[k],
				Options: v,
				Option:  v[0],
				Visible: true,
				Locale:  in.Language,
			},
		)
	}

	var singleAttributes = map[string]string{
		"Gender": in.Gender,
		"Store":  out.ActiveStore,
		"Brand":  in.Brand,
	}

	var (
		option  string
		options []string
	)
	for k := range singleAttributes {
		option = singleAttributes[k]
		if len(option) == 0 {
			continue
		}

		options = []string{option}

		_, exist = mappings.attributeMap[k]
		if !exist {
			log.Printf("Not found in attribute map - %s", k)
			continue
		}

		out.Attributes = append(
			out.Attributes,
			gwc.Attribute{
				Name:    k,
				ID:      *mappings.attributeMap[k],
				Options: options,
				Option:  option,
				Visible: true,
			},
		)
	}

	l := len(singleAttributes) + len(multiAttributes)
	attributes := make([]string, l)
	idx := 0
	for k0 := range singleAttributes {
		attributes[idx] = k0
		idx++
	}
	for k1 := range multiAttributes {
		attributes[idx] = k1
		idx++
	}

	return attributes, nil
}

// GetWCCategories uses a mapping table of the structure {gender: {name: id}} to translate names into Woocommerce IDs
func GetWCCategories(providerCategories []feed.ProviderCategory, categoryMap map[string]map[string][]*int32, allowMultiCats bool) (wcCategories []int32, err error) {
	var (
		exist, matched bool
		name, gender   string
		added, failed  int
	)

	unique := make(map[int32]struct{})
	for idx := range providerCategories {
		name = strings.ToLower(providerCategories[idx].Name)
		gender = string(providerCategories[idx].Gender)

		_, exist = categoryMap[gender][name]
		if !exist {
			CatNameMap := make(map[string]*string, len(categoryMap))
			for k := range categoryMap {
				key1 := k
				CatNameMap[key1] = &key1
			}
			name, matched = c.FuzzyFindReplace(name, CatNameMap)
			if !matched {
				failed++
				continue
			}
		}

		for j := range categoryMap[gender][name] {
			key2 := *categoryMap[gender][name][j]
			if key2 == 0 {
				failed++
				continue
			}
			_, exist = unique[key2]
			if exist {
				continue
			}
			unique[key2] = struct{}{}
			added++
		}
	}
	if added == 0 {
		return wcCategories, fmt.Errorf("No categories created for %v", providerCategories)
	}

	var it int
	if allowMultiCats {
		wcCategories = make([]int32, len(unique))
		for k := range unique {
			wcCategories[it] = k
			it++
		}
		return wcCategories, nil
	}

	var candidate int32
	for k := range unique {
		if it == 0 {
			candidate = k
			continue
		}
		if k != 2859 && k != 2854 {
			candidate = k
		}
	}
	return []int32{candidate}, nil
}
