package wooclient

import (
	"fmt"
	"hash/fnv"
)

// Tag interacts witht underlying category tree
type Tag struct {
	ID   int32  `json:"id,omitempty"`
	Name string `json:"name,omitempty"` // read-only
	Slug string `json:"slug,omitempty"` // read-only
}

// Dimension stores length, width, and height
type Dimension struct {
	Length string `json:"length,omitempty"`
	Width  string `json:"width,omitempty"`
	Height string `json:"height,omitempty"`
}

// WpmlPrice holds custom prices
type WpmlPrice struct {
	RegularPrice string `json:"regular_price,omitempty"`
	SalePrice    string `json:"sale_price,omitempty"`
}

// Product is the struct through which you interface with the WooCommerce backend
type Product struct {
	ID uint64 `json:"id,omitempty"` // read-only!!!
	//Key               uint64                   `json:"-"`
	SKU       string `json:"sku,omitempty"`
	Name      string `json:"name,omitempty"`
	Slug      string `json:"slug,omitempty"`
	Permalink string `json:"permalink,omitempty"` // read-only
	//DateCreated       string                   `json:"date_created,omitempty"`      // read-only
	DateCreatedGmt string `json:"date_created_gmt,omitempty"` // read-only
	//DateModified      string                   `json:"date_modified,omitempty"`     // read-only
	DateModifiedGmt   string `json:"date_modified_gmt,omitempty"` // read-only
	Type              string `json:"type"`
	Status            string `json:"status,omitempty"`
	Featured          bool   `json:"featured,omitempty"`
	CatalogVisibility string `json:"catalog_visibility,omitempty"` // Options: visible, catalog, search and hidden. Default is visible.
	Description       string `json:"description,omitempty"`
	ShortDescription  string `json:"short_description,omitempty"`
	//Price             string                   `json:"price,omitempty"`         // read-only
	RegularPrice      string `json:"regular_price,omitempty"`
	SalePrice         string `json:"sale_price,omitempty"`
	DateOnSaleFrom    string `json:"date_on_sale_from,omitempty"`
	DateOnSaleFromGmt string `json:"date_on_sale_from_gmt,omitempty"`
	DateOnSaleTo      string `json:"date_on_sale_to,omitempty"`
	DateOnSaleToGmt   string `json:"date_on_sale_to_gmt,omitempty"`
	//PriceHTML         string                   `json:"price_html,omitempty"`   // read-only
	OnSale bool `json:"on_sale,omitempty"` // read-only
	//Purchasable       bool                     `json:"purchasable,omitempty"`  // read-only
	TotalSales        string                   `json:"-"`                     // read-only
	ExternalURL       string                   `json:"external_url"`          // real outlink
	ButtonText        string                   `json:"button_text,omitempty"` // external shop to link to
	TaxStatus         string                   `json:"tax_status,omitempty"`  // Options: taxable, shipping and none. Default is taxable
	TaxClass          string                   `json:"tax_class,omitempty"`
	StockQuantity     int32                    `json:"stock_quantity,omitempty"`
	StockStatus       string                   `json:"stock_status,omitempty"`      // Options: instock, outofstock, onbackorder. Default is instock.
	SoldIndividually  bool                     `json:"sold_individually,omitempty"` // Allow one item to be bought in a single order. Default is false
	Weight            string                   `json:"weight,omitempty"`
	Dimensions        Dimension                `json:"dimensions,omitempty"`
	ShippingRequired  bool                     `json:"shipping_required,omitempty"` // read-only
	ReviewsAllowed    bool                     `json:"reviews_allowed,omitempty"`   // default: true
	AverageRating     string                   `json:"average_rating,omitempty"`    // read-only
	RatingCount       int32                    `json:"rating_count,omitempty"`      // read-only
	RelatedIds        []int32                  `json:"related_ids,omitempty"`       // read_only
	UpsellIds         []int32                  `json:"upsell_ids,omitempty"`
	CrossSellIds      []string                 `json:"cross_sell_ids,omitempty"`
	ParentID          int32                    `json:"parent_id,omitempty"`
	Categories        []Category               `json:"categories,omitempty"`
	Tags              []Tag                    `json:"tags,omitempty"`
	Images            []Image                  `json:"images,omitempty"`
	DefaultAttributes []map[string]interface{} `json:"default_attributes,omitempty"`
	Variations        []string                 `json:"variations,omitempty"`
	GroupedProducts   []int32                  `json:"grouped_products,omitempty"`
	MenuOrder         int32                    `json:"menu_order,omitempty"`
	MetaData          []map[string]interface{} `json:"meta_data,omitempty"`
	Attributes        []Attribute              `json:"attributes,omitempty"`
	Brands            []interface{}            `json:"brands,omitempty"`
	Language          string                   `json:"language,omitempty"`
	Lang              string                   `json:"lang,omitempty"`          // relates to the woocommerce multilingual package; otherwise: omit!
	CustomPrices      map[string]WpmlPrice     `json:"custom_prices,omitempty"` // "custom_prices": {"EUR": {"regular_price": 100, "sale_price": 99}}
}

// GetID implements Item
func (p Product) GetID() int32 {
	return int32(p.ID)
}

// GetKey returns a hash key based on name and sku
// to avoid dealing with the ID field (which is mandatory for updates but not allowed to be filled for creations)
func (p Product) GetKey() int32 {
	if p.SKU == "" && p.Name == "" {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(p.SKU + p.Name))

	return int32(h.Sum32())
}

// AddImage adds an image +name ( +text to display when said image not available)
func (p *Product) AddImage(url string, name string, text string) error {
	if url == "" || name == "" || text == "" {
		return fmt.Errorf("Can't add image wih empty fields")
	}
	p.Images = append(
		p.Images,
		Image{
			SRC:  url,
			Name: name,
			Alt:  text,
		},
	)
	return nil
}

// AddCategory adds category ID
func (p *Product) AddCategory(ID int32, name string) error {
	if ID == 0 {
		return fmt.Errorf("Can't add zero category id")
	}
	for i := range p.Categories {
		if p.Categories[i].ID == ID {
			return nil
		}
	}
	p.Categories = append(
		p.Categories,
		Category{
			ID:   ID,
			Name: name,
		},
	)

	return nil
}

// AddCategories adds multiple category IDs
func (p *Product) AddCategories(IDs []int32) error {
	var added bool
	for _, ID := range IDs {
		if ID == 0 {
			continue
		}
		for i := range p.Categories {
			if p.Categories[i].ID == ID {
				continue
			}
		}
		p.Categories = append(
			p.Categories,
			Category{
				ID: ID,
			},
		)
		added = true
	}
	if !added {
		return fmt.Errorf("No IDs added")
	}
	return nil
}
