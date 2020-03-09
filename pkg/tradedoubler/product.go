package tradedoubler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	c "stillgrove.com/gofeedyourself/pkg/collection"
	f "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	gtd "stillgrove.com/gofeedyourself/pkg/tradedoubler/client"
)

var (
	// Locales maps tradedoubler language names to WP locales
	Locales = map[string]string{
		"sv":    "sv_se",
		"sv_se": "sv_se",
	}
)

// Product wraps around tradedoubler products and adds feed methods
type Product struct {
	gtd.Product
	m *feed.Mapping
}

// ToFeedProduct returns pointer to a converted FeedProduct
func (p *Product) ToFeedProduct() (productOut *f.Product, err error) {
	err = p.Validate()
	if err != nil {
		return productOut, fmt.Errorf("Validate TD Product before conversion - %v", err)
	}

	productOut = &f.Product{
		Name:             c.Sanitize(strings.Replace(p.Name, p.Brand, "", -1)), // Make sure we don't get brand names mixed into the name
		Description:      c.Sanitize(p.Description),
		ShortDescription: c.Sanitize(p.ShortDescription),
		Brand:            c.Sanitize(p.Brand),
		ImageURL:         p.ProductImage.URL,
		Language:         p.Language,
	}

	lang, exist := Locales[p.Language]
	if !exist {
		return productOut, fmt.Errorf("Locale unknown - %s", p.Language)
	}
	productOut.Language = lang

	if len(p.m.ConversionMap) > 0 {
		uid := p.ID

		_, exist := p.m.ConversionMap[uid]
		if exist == true {
			productOut.Commision7d = p.m.ConversionMap[uid].Commision7d
			productOut.Conversions7d = p.m.ConversionMap[uid].Conversions7d
			productOut.Leads7d = p.m.ConversionMap[uid].Leads7d

			delete(p.m.ConversionMap, uid)
		}
	}

	for k := range p.Identifiers {
		if strings.ToLower(k) == "sku" {
			p.SKU = p.Identifiers[k]
		}
	}

	if p.FeedID == 0 {
		p.FeedID = 888
	}
	productOut.FromFeeds = append(productOut.FromFeeds, int32(p.FeedID))

	productOut.FromPrograms = append(productOut.FromPrograms, p.ProgramName)

	v := struct {
		multisizes   []string
		multicolors  []string
		multipattern []string
		multicat     []string
		gender       []string
		catPath      string
		col          string
	}{
		multicat: []string{
			p.Name,
		},
	}

	var hasSize bool
	for _, f := range p.Fields {
		switch name := strings.ToLower(f["name"].(string)); name {
		/*case "item_group_id":
		productOut.TdCategoryID = f["value"].(string)*/
		case "gender":
			productOut.Gender = strings.ToLower(f["value"].(string))
			//v.gender = append(v.gender, strings.ToLower(f["value"].(string)))
		case "color":
			v.col = c.Sanitize(f["value"].(string))
			arr := c.SplitList(f["value"].(string))
			for i := range arr {
				v.multicolors = append(v.multicolors, c.Sanitize(arr[i]))
			}
		case "colour":
			v.col = c.Sanitize(f["value"].(string))
			arr := c.SplitList(f["value"].(string))
			for i := range arr {
				v.multicolors = append(v.multicolors, c.Sanitize(arr[i]))
			}
		/*case "colors":
			multicolors = append(multicolors, f["value"].(string))
		case "colours":
			multicolors = append(multicolors, f["value"].(string))*/
		case "sizes":
			hasSize = true
			if len(c.SplitList(f["value"].(string))) > len(v.multisizes) {
				v.multisizes = c.SplitList(f["value"].(string))
			}
		case "size":
			hasSize = true
			if len(c.SplitList(f["value"].(string))) > len(v.multisizes) {
				v.multisizes = c.SplitList(f["value"].(string))
			}
		case "main category":
			v.multicat = append(
				v.multicat,
				strings.ToLower(f["value"].(string)),
			)
		case "subcategory":
			v.multicat = append(
				v.multicat,
				strings.ToLower(f["value"].(string)),
			)
			v.catPath = c.CollateString(v.catPath, f["value"].(string))
			v.gender = append(v.gender, f["value"].(string))
		case "subcategorypath":
			v.multicat = append(
				v.multicat,
				strings.ToLower(f["value"].(string)),
			)
			v.catPath = c.CollateString(v.catPath, f["value"].(string))
			v.gender = append(v.gender, f["value"].(string))
		case "material":
			productOut.Material = f["value"].(string)
		case "item_group_id":
			p.ItemGroupID = f["value"].(string)
		case "gtin":
			p.GTIN = f["value"].(string)
		}
	}

	v.multicolors = append(
		v.multicolors,
		productOut.Name,
	)

	productOut.Color = v.col
	productOut.ColorGroups, err = c.MapAttributes(v.multicolors, p.m.ColorMap, "", true)
	if err != nil {
		return productOut, err
	}

	v.multipattern, err = c.MapAttributes(v.multicolors, p.m.PatternMap, "Various", true)
	if err != nil {
		return productOut, fmt.Errorf("To Feed Product - Pattern - %v", err)
	}

	if hasSize {
		v.multisizes, err = c.MapAttributes(v.multisizes, p.m.SizeMap, "", false)
		if err != nil {
			return productOut, fmt.Errorf("To Feed Product - Sizes - %v", err)
		}
	}

	if !hasSize || c.StringInList("onesize", v.multisizes) {
		v.multisizes = []string{"onesize"}
	}

	productOut.OriginalCategories = []string{v.catPath}
	for i := range p.Categories {
		v.multicat = append(v.multicat, strings.ToLower(p.Categories[i].Name))
	}
	productOut.ProviderCategories, err = p.processCategories(v.multicat, strings.ToLower(productOut.Gender))
	if v.catPath == "" && err != nil {
		return productOut, fmt.Errorf("To Feed Product - Categories - %v", err)
	}

	productOut.Patterns = c.UniqueNames(v.multipattern)

	shippingCostRules := NewShippingCostRuleSet()
	shippingCostRules.Add(25437, "SEK", 400, 39.90)
	productOut.Retailers, productOut.RetailerMap, err = processOffers(p.Offers, v.multisizes, shippingCostRules)
	if err != nil {
		return productOut, fmt.Errorf("Failed to process offers for %s - %v", productOut.Name, err)
	}

	for i := range productOut.Retailers {
		p, _ := strconv.ParseFloat(productOut.Retailers[i].Price, 32)
		if float32(p) < productOut.HighestPrice {
			productOut.HighestPrice = float32(p)
		}
		if float32(p) > productOut.HighestPrice {
			productOut.HighestPrice = float32(p)
		}
	}

	productOut.SKU = c.CollateStrings(
		p.SKU,
		p.GTIN,
		p.ItemGroupID,
	)
	if p.FeedID == 25437 {
		productOut.SKU = p.ItemGroupID
	}

	productOut.SetKey()
	if productOut.Key == 0 {
		return productOut, fmt.Errorf("Couldn't set key for %s", p.Name)
	}

	err = productOut.CalculateDiscounts(10)
	if err != nil {
		return productOut, fmt.Errorf("Convert TD Product - Calculate Prices - %v", err)
	}

	err = productOut.Validate()
	if err != nil {
		return productOut, fmt.Errorf("Convert TD Product - %v", err)
	}

	productOut.Active = true

	return productOut, nil
}

// Validate checks for incosnistencies in a product
func (p *Product) Validate() (err error) {
	if p.Name == "" {
		return fmt.Errorf("Product doesn't have a name")
	}

	desc := c.CollateString(p.Description, p.ShortDescription)
	var checks = map[string]*string{
		"Name":          &p.Name,
		"Description":   &desc,
		"Brand":         &p.Brand,
		"Language":      &p.Language,
		"Product Image": &p.ProductImage.URL,
	}
	for k, v := range checks {
		val := *v
		if val == "" {
			return fmt.Errorf("Tradedoubler Feed - Product %s is missing", k)
		}
	}

	err = p.validateFields()
	if err != nil {
		return fmt.Errorf("Tradedoubler Feed - %v", err)
	}

	_, e0 := p.Identifiers["sku"]
	_, e1 := p.Identifiers["SKU"]

	var e2 bool
	for ix := range p.Fields {
		if p.Fields[ix]["name"] == "item_group" || p.Fields[ix]["name"] == "gtin" {
			e2 = true
			break
		}
	}

	if !e0 && !e1 && !e2 {
		return fmt.Errorf("No SKU/gtin found")
	}

	var anyPrices, anyOffers bool
	for i := range p.Offers {
		if c.AnyEmpty(
			[]*string{
				&p.Offers[i].Availability,
				&p.Offers[i].ProductURL,
			},
		) {
			continue
		}

		for j := range p.Offers[i].PriceHistory {
			_, e3 := p.Offers[i].PriceHistory[j].Price["value"]
			_, e4 := p.Offers[i].PriceHistory[j].Price["value"]

			if e3 && e4 {
				anyPrices = true
			}
		}
		anyOffers = true
	}
	if !anyOffers || !anyPrices {
		return fmt.Errorf("No valid offers")
	}

	return nil
}

func (p *Product) validateFields() error {
	if len(p.Fields) == 0 {
		return fmt.Errorf("No fields in td product")
	}
	var names []string
	for i := range p.Fields {
		if p.Fields[i]["name"] == nil {
			return fmt.Errorf("Field has no name - %v", p.Fields)
		}
		names = append(names, strings.ToLower(p.Fields[i]["name"].(string)))
		if p.Fields[i]["value"] == nil {
			return fmt.Errorf("Field has no value - %v", p.Fields)
		}
	}
	if !c.ListInList(
		names,
		[]string{
			"color",
			"colour",
			"colors",
			"colours",
		},
	) {
		return fmt.Errorf("Missing color field - %s", p.Name)
	}
	/*if !c.ListInList(
		names,
		[]string{
			"size",
			"sizes",
		},
	) {
		return fmt.Errorf("Missing size field - %s", p.Name)
	}*/
	return nil
}

func (p *Product) processCategories(candidates []string, gender string) (outCats []f.ProviderCategory, err error) {
	vars := struct {
		matched  bool
		levels   []string
		cName    string
		inGender rune
		cat      f.ProviderCategory
	}{}

	switch g := strings.ToLower(gender); g {
	case "m":
		vars.inGender = 'm'
	case "men":
		vars.inGender = 'm'
	case "male":
		vars.inGender = 'm'
	case "w":
		vars.inGender = 'w'
	case "women":
		vars.inGender = 'w'
	case "f":
		vars.inGender = 'w'
	case "female":
		vars.inGender = 'w'
	default:
		vars.inGender = 'u'
	}

	var markers = map[string]rune{
		"women":  'w',
		"female": 'w',
		"men":    'm',
		"male":   'm',
	}

	// Reconstruct generic gender categories -couldn't do better
	/*var genderTerm string
	if vars.inGender == 'm' {
		genderTerm = "men"
	}
	if vars.inGender == 'w' {
		genderTerm = "women"
	}*/

	categories, err := c.MapAttributes(candidates, p.m.CatNameMap, "", true)
	if err != nil {
		return outCats, err
	}

	var longestMatch int
	var name string
	for _, v := range categories {
		name = strings.ToLower(c.Sanitize(v))
		if vars.inGender != 'm' && vars.inGender != 'w' {
			vars.matched = false
			longestMatch = 0
			for k := range markers {
				vars.matched, _ = regexp.MatchString(k+".*", name)
				if vars.matched == true && len(k) > longestMatch {
					vars.inGender = markers[k]
					longestMatch = len(k)
				}
			}
		}

		if name == "men" && vars.inGender != 'm' {
			continue
		}
		if name == "women" && vars.inGender != 'w' {
			continue
		}

		vars.cat = f.ProviderCategory{
			ProviderName: "tradedoubler",
			//ProviderCategoryID: v.ID,
			Gender: vars.inGender, // can be: m, f, u
			Name:   name,
		}
		outCats = append(outCats, vars.cat)
	}
	if len(outCats) == 0 {
		return outCats, fmt.Errorf("Tradedoubler: Couldn't map categories for %v", p.Name)
	}

	return outCats, nil
}

func processOffers(offers []gtd.Offer, allSizes []string, rs ShippingCostRuleSet) (retailers []f.Retailer, retailerMap map[uint64]struct{}, err error) {
	if len(allSizes) == 0 {
		return retailers, retailerMap, fmt.Errorf("No sizes")
	}

	retailerMap = make(map[uint64]struct{})

	for i := range offers {
		var r = f.Retailer{
			Link:         offers[i].ProductURL,
			Logo:         offers[i].ProgramLogo,
			Name:         c.Sanitize(strings.Replace(offers[i].ProgramName, ".com", "", -1)),
			DeliveryTime: offers[i].DeliveryTime,
			ShippingCost: offers[i].ShippingCost,
			Sizes:        allSizes,
		}

		avail := c.Sanitize(strings.ToLower(offers[i].Availability))
		if avail == "yes" || avail == "in stock" || avail == "instock" {
			r.Availability = "instock"
		} else {
			continue
		}

		// reagardless of availability I want to check for lowest and highest prices, after all: a discount is a discount
		price := struct {
			cp32       float32
			cp64       float64
			mostRecent int32
			highest    float32
		}{}

		for n, h := range offers[i].PriceHistory {
			price.cp64, _ = strconv.ParseFloat(h.Price["value"], 32)
			price.cp32 = float32(price.cp64)
			if n == 0 {
				price.highest = price.cp32

				price.mostRecent = int32(h.Date)
				r.Price = h.Price["value"]
				r.Currency = h.Price["currency"]
			}
			if int32(h.Date) > price.mostRecent {
				price.mostRecent = int32(h.Date)
				r.Price = h.Price["value"]
				r.Currency = h.Price["currency"]
			}
			if price.cp32 > price.highest {
				price.highest = price.cp32
			}
		}

		// Apply shipping cost rules
		if r.ShippingCost == "" {
			p, err := strconv.ParseFloat(r.Price, 32)
			if err == nil {
				r.ShippingCost, _ = rs.GetPrice(r.Currency, offers[i].FeedID, float32(p))
			}
		}

		r.HighestPrice = price.highest

		retailers = append(retailers, r)
		retailerMap[c.HashKey(r.Link)] = struct{}{}
	}

	if len(retailers) == 0 {
		return retailers, retailerMap, fmt.Errorf("No active offers")
	}

	return retailers, retailerMap, nil
}
