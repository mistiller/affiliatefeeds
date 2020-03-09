package feed

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"stillgrove.com/gofeedyourself/pkg/collection"
	c "stillgrove.com/gofeedyourself/pkg/collection"
)

var (
	// Locales is a whitelist of known locales
	Locales = []string{
		"sv_se",
	}
)

// Retailer captures (multiple) active vendors for one product
type Retailer struct {
	Link         string  `json:"link"`
	Logo         string  `json:"logo"`
	Name         string  `json:"name"`
	Price        string  `json:"price"`
	HighestPrice float32 `json:"highestPrice"`
	//DisplayPrice string `json:"displyPrice"` // specific thing for Boozt - only shows up in CSV feeds?
	Currency     string `json:"currency"`
	Availability string `json:"availability"`
	DeliveryTime string `json:"deliveryTime"`
	ShippingCost string `json:"shippingCost"`
	IsCrawler    bool
	Sizes        []string `json:"sizes,omitempty"`
}

type attribute struct {
	ID      int    `json:"id"`
	Options string `json:"options"`
}

//ProviderCategory caputured source category information
// before we transform it to the destination format
type ProviderCategory struct {
	ProviderName       string `json:"providerName"`
	ProviderCategoryID int    `json:"id"`
	Name               string `json:"name"`   // mandatory!
	Gender             rune   `json:"gender"` // mandatory!
}

// Product is the common currency of entites loaded from the network feeds
type Product struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	ShortDescription string     `json:"shortDescription"`
	Brand            string     `json:"brand"`
	Retailers        []Retailer `json:"retailers"`
	ImageURL         string     `json:"image_url"`
	HighestPrice     float32    `json:"highest_price"`
	LowestPrice      float32    `json:"lowest_price"`
	Discount         int        `json:"discount"`
	DiscountBins     []string   `json:"discount_bins"`
	//TdCategoryID     string     `json:"category_id"`
	//TdCategoryList     []string
	//TdCategoryLvl1     []string
	Key             uint64   `json:"key"`
	SKU             string   // just string conversion of key to later store in woocommerce backend
	Active          bool     `json:"active"`
	LastSeen        int32    `json:"lastSeen"`
	Color           string   `json:"color"`
	ColorGroups     []string `json:"colorGroup"` // trying to make colors searchable via grouping
	Patterns        []string `json:"pattern"`
	Material        string   `json:"material"`
	Gender          string   `json:"gender"`
	Language        string   `json:"language"`
	WebsiteFeatures int32    //`json:"website_features"`
	ExpectedValue   float32
	Commision7d     float32  //`json:"-"`
	Leads7d         int32    //`json:"-"`
	Conversions7d   int32    //`json:"-"`
	FromFeeds       []int32  `json:"fromFeeds"`
	FromPrograms    []string `json:"fromPrograms"`
	Categories      []int32
	//WCAttributes    []attribute
	//WCCategories       []int32
	ProviderCategories []ProviderCategory  // stores the feed provider's category information as generically as possible
	OriginalCategories []string            // stores the original category
	RetailerMap        map[uint64]struct{} // helps to keep the reatilers unique
}

// GetKey returns the internal common key
func (p *Product) GetKey() uint64 {
	return p.Key
}

// SetKey implementes the Product interface
func (p *Product) SetKey() error {
	var name string
	if len(p.SKU)*len(p.Color) == 0 {
		return fmt.Errorf("Missing SKU or color to construct ID")
	}
	name = p.SKU + "-" + c.SanitizeHard(p.Color)

	h := fnv.New64a()
	h.Write([]byte(name))

	p.Key = h.Sum64()

	return nil
}

func (p *Product) AsRows() (header []string, rows [][]string, err error) {
	err = p.CalculateDiscounts(10)
	if err != nil {
		return header, rows, err
	}

	header = []string{
		"Name",
		"SKU",
		"Description",
		"Brand",
		"ExtractedColors",
		"OriginalColor",
		"Gender",
		"ImageURL",
		"ExtractedCategories",
		"OriginalCategory",
		"Store",
		"Availability",
		"StoreLink",
		"RegularPrice",
		"SalesPrice",
		"Discount",
		"DiscountBins",
		"Size",
		"Delivery",
		"Shipping Cost",
		"LastWeeksConversions",
	}

	var categories []string
	for c := range p.ProviderCategories {
		if p.ProviderCategories[c].Name == "" {
			continue
		}
		categories = append(categories, p.ProviderCategories[c].Name)
	}

	var (
		sizes        string
		colors       string
		salesPrice   string
		regularPrice string
		cats         string
		discount     int
		discountBins []string
	)

	colors = strings.Trim(
		strings.Join(p.ColorGroups, ","),
		",",
	)
	cats = strings.Trim(
		strings.Join(categories, ","),
		",",
	)

	if !collection.StringInList(
		p.Gender,
		[]string{
			"women",
			"men",
			"unisex",
		},
	) {
		return header, rows, fmt.Errorf("Product dows not qualify, gender is %s", p.Gender)
	}

	for i := range p.Retailers {
		if p.Retailers[i].Availability != "instock" {
			continue
		}
		sizes = strings.Trim(
			strings.Join(p.Retailers[i].Sizes, ","),
			",",
		)

		price, _ := strconv.ParseFloat(p.Retailers[i].Price, 64)
		if float32(price) < p.HighestPrice {
			salesPrice = p.Retailers[i].Price
			regularPrice = fmt.Sprintf("%f", p.HighestPrice)

			discount, discountBins = getDiscountBins(float32(price), p.HighestPrice, 10)
		} else {
			regularPrice = p.Retailers[i].Price
		}

		rows = append(
			rows,
			[]string{
				p.Name,
				p.SKU,
				collection.CollateStrings(p.Description, p.ShortDescription),
				p.Brand,
				colors,
				p.Color,
				p.Gender,
				p.ImageURL,
				cats,
				collection.CollateStrings(p.OriginalCategories...),
				p.Retailers[i].Name,
				p.Retailers[i].Availability,
				p.Retailers[i].Link,
				regularPrice,
				salesPrice,
				fmt.Sprintf("%d", discount), // discount
				strings.Trim(
					strings.Join(discountBins, ","),
					",",
				), // discountBins
				sizes,
				p.Retailers[i].DeliveryTime,
				p.Retailers[i].ShippingCost,
				fmt.Sprintf("%d", p.Conversions7d),
			},
		)
	}

	if len(rows) == 0 {
		return header, rows, fmt.Errorf("No eligible rows for product")
	}

	return header, rows, nil
}

// Update makes sure that we always have a proper key, timestamp, and size-array
func (p *Product) Update() (err error) {
	if p.Key == 0 {
		p.SetKey()
	}
	err = p.CalculateDiscounts(10)
	if err != nil {
		return fmt.Errorf("Recalculate discounts - %s - %v", p.Name, err)
	}

	/*if len(p.ColorGroups) != 0 {
		for i := range p.ColorGroups {
			p.Color = p.ColorGroups[i]
		}
	}*/

	p.Active = false
	for j := range p.Retailers {
		if p.Retailers[j].IsCrawler {
			continue
		}
		if p.Retailers[j].Availability == "instock" {
			p.Active = true
		}
	}

	if len(p.Retailers) != len(p.RetailerMap) {
		var (
			key uint64
			ex  bool
		)
		for i := range p.Retailers {
			key = c.HashKey(p.Retailers[i].Link)
			_, ex = p.RetailerMap[key]
			if !ex {
				p.RetailerMap[key] = struct{}{}
			}
		}
	}

	p.LastSeen = int32(time.Now().Unix())

	return nil
}

// MergeWith allows you to consolidate two products with the same id
// by merging the new information from newProduct into p
func (p *Product) MergeWith(newProduct *Product) error {
	p.Name = c.CollateString(p.Name, newProduct.Name)
	p.Description = c.CollateString(p.Description, newProduct.Description)
	p.ShortDescription = c.CollateString(p.ShortDescription, newProduct.ShortDescription)
	p.Brand = c.CollateString(p.Brand, newProduct.Brand)

	p.ImageURL = c.CollateString(p.ImageURL, newProduct.ImageURL)

	p.HighestPrice = c.HighestFloat(
		[]float32{
			p.HighestPrice,
			newProduct.HighestPrice,
		},
	)
	p.LowestPrice = c.HighestFloat(
		[]float32{
			p.LowestPrice,
			newProduct.LowestPrice,
		},
	)
	if p.HighestPrice == 0.0 {
		p.HighestPrice = p.LowestPrice
	}
	if p.LowestPrice == 0.0 {
		p.LowestPrice = p.HighestPrice
	}

	p.SKU = c.CollateString(p.SKU, newProduct.SKU)
	p.Gender = c.CollateString(p.Gender, newProduct.Gender)

	p.Color = c.CollateString(p.Color, newProduct.Color)
	p.ColorGroups = c.MergeLists(p.ColorGroups, newProduct.ColorGroups)
	p.Patterns = c.MergeLists(p.Patterns, newProduct.Patterns)

	retailerMap := make(map[uint64]struct{})
	retailerMap = p.RetailerMap
	var hashKey uint64
	for j := range newProduct.Retailers {
		hashKey = c.HashKey(newProduct.Retailers[j].Link)

		_, exist := retailerMap[hashKey]
		if exist == false {
			p.Retailers = append(p.Retailers, newProduct.Retailers[j])
			retailerMap[hashKey] = struct{}{}
		}
	}
	p.RetailerMap = retailerMap
	retailerMap = nil

	err := p.CalculateDiscounts(10)
	if err != nil {
		return fmt.Errorf("Merging products - %v", err)
	}

	for i := range newProduct.ProviderCategories {
		p.ProviderCategories = append(p.ProviderCategories, newProduct.ProviderCategories[i])
	}

	p.WebsiteFeatures += newProduct.WebsiteFeatures

	p.Leads7d += newProduct.Leads7d
	p.Conversions7d += newProduct.Conversions7d
	p.Commision7d += newProduct.Commision7d

	for i := range newProduct.FromPrograms {
		p.FromPrograms = append(p.FromPrograms, newProduct.FromPrograms[i])
	}
	for i := range newProduct.FromFeeds {
		p.FromFeeds = append(p.FromFeeds, newProduct.FromFeeds[i])
	}

	err = p.Update()
	if err != nil {
		return fmt.Errorf("Update after merge - %v", err)
	}

	return nil
}

// CalculateRanking is where we apply the formula to rank products
func (p *Product) CalculateRanking() int32 {
	w := struct {
		leads       int32
		conversions int32
		features    int32
	}{
		leads:       2,
		conversions: 5,
		features:    10,
	}

	//commissionWeight := 20

	return p.WebsiteFeatures*w.features + p.Leads7d*w.leads + p.Conversions7d*w.conversions //+ p.Commision7d*commissionWeight
}

// CalculateDiscounts looks updates the values for discounts based on current lowest and highest prices
func (p *Product) CalculateDiscounts(binSize int) (err error) {
	if binSize < 1 {
		binSize = 10
	}

	if p.HighestPrice*p.LowestPrice == 0.0 {
		var (
			prices []float32
			p64    float64
		)

		for i := range p.Retailers {
			p64, err = strconv.ParseFloat(p.Retailers[i].Price, 32)
			if err != nil {
				continue
			}
			prices = append(prices, float32(p64))
			if p.Retailers[i].HighestPrice > float32(p64) {
				prices = append(prices, p.Retailers[i].HighestPrice)
			}
		}

		if p.HighestPrice == 0.0 {
			p.HighestPrice = c.HighestFloat(prices)
		}
		if p.LowestPrice == 0.0 {
			p.LowestPrice = c.LowestFloat(prices)
		}
	}

	if p.HighestPrice < p.LowestPrice {
		p.HighestPrice = p.LowestPrice
	}
	if p.LowestPrice > p.HighestPrice {
		p.LowestPrice = p.HighestPrice
	}

	if p.LowestPrice != p.HighestPrice {
		/*p.Discount = int(((p.LowestPrice-p.HighestPrice)/p.HighestPrice)*100) * -1
		if p.Discount < binSize {
			return nil
		}

		p.DiscountBins = []string{}
		var bin string
		it := 1
		for idx := binSize; idx < p.Discount; idx += binSize {
			bin = fmt.Sprint(it*binSize) + "%"
			p.DiscountBins = append(p.DiscountBins, bin)
			it++
		}*/

		p.Discount, p.DiscountBins = getDiscountBins(p.LowestPrice, p.HighestPrice, binSize)
	}

	return nil
}

func getDiscountBins(lowestPrice, highestPrice float32, binSize int) (discount int, discountBins []string) {
	discount = int(((lowestPrice-highestPrice)/highestPrice)*100) * -1
	if discount < binSize {
		return discount, discountBins
	}

	var bin string
	it := 1
	for idx := binSize; idx < discount; idx += binSize {
		bin = fmt.Sprint(it*binSize) + "%"
		discountBins = append(discountBins, bin)
		it++
	}

	return discount, discountBins
}

// Validate makes sure that we have a complete product
func (p *Product) Validate() (err error) {
	if p.Active == false {
		return nil
	}

	description := c.CollateString(p.Description, p.ShortDescription)
	var checkMap = map[string]*string{
		"Name":        &p.Name,
		"SKU":         &p.SKU,
		"Color":       &p.Color,
		"Description": &description,
		"Gender":      &p.Gender,
		"Brand":       &p.Brand,
		"Language":    &p.Language,
		"Image":       &p.ImageURL,
	}
	for k, v := range checkMap {
		if c.IsEmpty(v) {
			return fmt.Errorf("Validate Feed Product - %s missing - %s", k, p.Name)
		}
	}

	if !c.StringInList(p.Language, Locales) {
		return fmt.Errorf("Validate Feed Product - Unknown Language - %s", p.Language)
	}

	/*if len(p.ColorGroups) == 0 {
		return fmt.Errorf("Validate Feed Product - Color not mapped - %s", p.Color)
	}*/

	err = p.validateRetailers()
	if err != nil {
		return fmt.Errorf("Validate Feed Product - %v", err)
	}
	err = p.validatePrices()
	if err != nil {
		return fmt.Errorf("Validate Feed Product - %v", err)
	}

	if len(p.ProviderCategories)+len(p.OriginalCategories) == 0 {
		return fmt.Errorf("Validate Feed Product - Categories missing - %s", p.Name)
	}

	for i := range p.ProviderCategories {
		if p.ProviderCategories[i].Gender == 0 ||
			p.ProviderCategories[i].Name == "" {
			return fmt.Errorf("Validate Feed Product - Category Incomplete")
		}
	}

	return nil
}

func (p *Product) validateRetailers() error {
	if len(p.RetailerMap) != len(p.Retailers) {
		return fmt.Errorf("Retailer Map inconsistent - %s - %v", p.Name, p.Retailers)
	}

	for idx := 0; idx < len(p.RetailerMap); idx++ {
		if c.AnyEmpty(
			[]*string{
				&p.Retailers[idx].Link,
				&p.Retailers[idx].Availability,
			}) {
			return fmt.Errorf("Retailer fields missing")
		}
		if len(p.Retailers[idx].Sizes) == 0 {
			return fmt.Errorf("Sizes missing for Retailer")
		}
		for i := range p.Retailers[idx].Sizes {
			if strings.Contains(p.Retailers[idx].Sizes[i], ",") {
				return fmt.Errorf("Sizes were not split correctly - %s", p.Retailers[idx].Sizes[i])
			}
		}
		idx++
	}
	return nil
}

func (p *Product) validatePrices() error {
	if p.HighestPrice*p.LowestPrice == 0 || p.HighestPrice < p.LowestPrice {
		return fmt.Errorf("Prices inconsistent")
	}
	if p.LowestPrice != p.HighestPrice &&
		(p.Discount == 0 || len(p.DiscountBins) == 0) {
		return fmt.Errorf("Prices and discounts inconsistent")
	}
	return nil
}
