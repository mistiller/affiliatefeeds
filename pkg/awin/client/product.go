package awinclient

import (
	"fmt"
	"hash/fnv"
	"strconv"

	"stillgrove.com/gofeedyourself/pkg/collection"
)

// Product holds product entries in the downloadable json files
// CAREFUL: Order is important when parsing json. Also: Never omitempty.
type Product struct {
	DataFeedID        string `json:"data_feed_id"` // mandatory!
	MerchantID        string `json:"merchant_id"`
	MerchantName      string `json:"merchant_name"` // mandatory!
	AWProductID       string `json:"aw_product_id"`
	AWDeepLink        string `json:"aw_deep_link"` // mandatory!
	AWImageURL        string `json:"aw_image_url"`
	AWThumpURL        string `json:"aw_thumb_url"`
	CategoryID        string `json:"category_id"`
	CategoryName      string `json:"category_name"` // mandatory!
	BrandID           string `json:"brand_id"`
	BrandName         string `json:"brand_name"` // mandatory!
	MerchantProductID string `json:"merchant_product_id"`
	MerchantCategory  string `json:"merchant_category"`
	EAN               string `json:"ean,omitempty"`
	ISBN              string `json:"isbn,omitempty"`
	ParentProductID   string `json:"parent_product_id"`
	GTIN              string `json:"product_GTIN"`
	ModelNumber       string `json:"model_number"`
	ProductName       string `json:"product_name"` // mandatory!
	Description       string `json:"description"`  // mandatory!
	Specifications    string `json:"specifications,omitempty"`
	PromotionalText   string `json:"promotional_text"`
	Language          string `json:"language"`
	MerchantDeepLink  string `json:"merchant_deep_link"`
	MerchantImageURL  string `json:"merchant_image_url"`
	DeliveryTime      string `json:"delivery_time,omitempty"`
	Currency          string `json:"currency"` // mandatory!
	SearchPrice       string `json:"search_price"`
	RPPPrice          string `json:"rpp_price"`
	DeliveryCost      string `json:"delivery_cost,omitempty"`
	InStock           string `json:"in_stock"`
	StockQuantity     string `json:"stock_quantity"`
	ProductType       string `json:"product_type"`
	Colour            string `json:"colour"` // mandatory!
	Custom1           string `json:"custom_1,omitempty"`
	Custom2           string `json:"custom_2,omitempty"`
	Reviews           string `json:"reviews"`

	Size             string `json:"Fashion:size"`
	Material         string `json:"Fashion:Material"`
	SuitableCategory string `json:"Fashion:suitable_for"`
	Category         string `json:"Fashion:category"`
	Pattern          string `json:"Fashion:pattern"`
	Swatch           string `json:"Fashion:swatch"`

	Rating         string `json:"rating"`
	AlternateImage string `json:"altenate_image"`

	ProductShortDescription string `json:"product_short_description"` // mandatory!

	StockStatus string `json:"stock_status"` // mandatory!
	ValidFrom   string `json:"valid_from"`
	ValidTo     string `json:"valid_to"` // mandatory!

	Keywords string `json:"keywords,omitempty"`

	BasePrice       string `json:"base_price"`
	BasePriceAmount string `json:"base_price_amount"`
	BasePriceText   string `json:"base_price_text"`
	ProductPriceOld string `json:"product_price_old"`
	DisplayPrice    string `json:"display_price"`

	SizeStockAmount string `json:"size_stock_amount,omitempty"`

	Custom3 string `json:"custom_3,omitempty"`
	Custom4 string `json:"custom_4,omitempty"`
	Custom5 string `json:"custom_5,omitempty"`
	Custom6 string `json:"custom_6,omitempty,omitempty"`
	Custom7 string `json:"custom_7,omitempty,omitempty"`
	Custom8 string `json:"custom_8,omitempty,omitempty"`
	Custom9 string `json:"custom_9,omitempty,omitempty"`

	ExpectedValue float64 `json:"expected_value,omitempty"`
}

func (p *Product) Key() (hash uint64, err error) {
	s := p.AWProductID + p.BrandID + p.ProductName
	if s == "" {
		return hash, fmt.Errorf("Couldn't create key")
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64(), nil
}

func (p *Product) AttachProgrammes(prog []Programme) error {
	var (
		id  int
		err error
	)
	for i := range prog {
		id, err = strconv.Atoi(p.MerchantID)
		if err != nil {
			return fmt.Errorf("Parse MerchantID - %v", err)
		}
		if prog[i].ProgrammeInfo.ID == uint64(id) {
			err = p.handleProgramme(&prog[i])
			if err != nil {
				return fmt.Errorf("Parse Programme - %v", err)
			}
		}
	}
	return nil
}

func (p *Product) handleProgramme(prog *Programme) error {
	var (
		approval, price, value float64
	)
	approval = prog.KPI.ApprovalPercentage
	price, _, _ = p.GetPrices()
	for i := range prog.CommissionGroups {
		if prog.CommissionGroups[i].Type == "percentage" {
			value = prog.CommissionGroups[i].Percentage * price
		} else if prog.CommissionGroups[i].Type == "fix" {
			value = prog.CommissionGroups[i].Amount
		} else {
			return fmt.Errorf("Failed to handle commission")
		}
	}

	if approval > 1 {
		approval /= 100.0
	}

	if p.ExpectedValue > 0.0 {
		p.ExpectedValue = ((value * approval) + p.ExpectedValue) / 2
	} else {
		p.ExpectedValue = value * approval
	}

	return nil
}

func (p *Product) GetPrices() (low, high float64, num int) {
	var (
		err   error
		price float64
		str   = [...]string{
			p.RPPPrice,
			p.SearchPrice,
			p.BasePrice,
			p.BasePriceAmount,
			p.BasePriceText,
			p.DisplayPrice,
		}
	)
	for i := range str {
		if str[i] == "" {
			continue
		}
		price, err = strconv.ParseFloat(collection.Sanitize(str[i]), 32)
		if err != nil {
			continue
		}
		if price == 0.0 {
			continue
		}
		if num == 0 {
			high = price
			low = high
			num++
		}
		if price == low && price == high {
			continue
		}
		if price < low {
			low = price
		}
		if price > high {
			high = price
		}
		num++
	}

	return low, high, num
}

func (p *Product) Validate() error {
	var strFields = map[string]string{
		"MerchantName": p.MerchantName,
		"AWDeepLink":   p.AWDeepLink,
		"BrandName":    p.BrandName,
		//"Language":     p.Language,
		"AWProductID": p.AWProductID,
		"DataFeedID":  p.DataFeedID,
		"ProductName": p.ProductName,
	}

	for k, v := range strFields {
		if v == "" {
			return fmt.Errorf("Validation - Awin Field missing - %s", k)
		}
	}

	if len(p.Description)+len(p.ProductShortDescription)+len(p.PromotionalText) == 0 {
		return fmt.Errorf("Validation - No description found - %s", p.ProductName)
	}

	if p.CategoryName == "" && p.MerchantCategory == "" {
		return fmt.Errorf("Validation - Neither CategoryName nor MerchantCategory set - %s", p.ProductName)
	}

	lowestPrice, highestPrice, nPrices := p.GetPrices()

	if lowestPrice*highestPrice == 0.0 {

		return fmt.Errorf("Validation - Highest and lowest prices must be set - %f - %f - %d", lowestPrice, highestPrice, nPrices)
	}
	if nPrices > 1 && lowestPrice >= highestPrice {
		return fmt.Errorf("Validation - Too many lowest price not lower highest")
	}
	if nPrices == 1 && lowestPrice != highestPrice {
		return fmt.Errorf("Validation - Lowest price and highest should be equal")
	}

	return nil
}
