package storefront

import (
	"encoding/json"
	"fmt"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

type ProductCategory struct {
	CategoryID uint64 `json:"category_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Path       string `json:"path"`
}

type MediaGallery struct {
	Image string `json:"image"` // see: Image
	Pos   uint64 `json:"pos"`   // 1
	Typ   string `json:"typ"`   // "image"
	Lab   string `json:"lab"`   // null
	Vid   string `json:"vid"`   // null
}

type Stock struct {
	IsInStock bool   `json:"is_in_stock"`
	Qty       uint64 `json:"qty"`
}

type Product struct {
	ID                  uint64            `json:"id"`
	Name                string            `json:"name"`
	Image               string            `json:"image"` //Local file!
	SKU                 string            `json:"sku"`
	URLKey              string            `json:"url_key"`  // slug
	URLPath             string            `json:"url_path"` // PDP Path
	TypeID              string            `json:"type_id"`  // configurable, simple, e.g.
	Price               float64           `json:"price"`
	SpecialPrice        float64           `json:"special_price"` // special = sale
	PriceInclTax        float64           `json:"price_incl_tax"`
	SpecialPriceInclTax float64           `json:"special_price_incl_tax"`
	SpecialToDate       string            `json:"special_to_date"`
	SpecialFromDate     string            `json:"special_from_date"`
	Status              uint64            `json:"status"`
	Visibility          uint64            `json:"visibility"`
	Size                uint64            `json:"size"`
	SizeOptions         []uint64          `json:"size_options"`
	Color               uint64            `json:"color"`
	ColorOptions        []uint64          `json:"color_options"`
	CategoryIDs         []string          `json:"category_ids"`
	Category            []ProductCategory `json:"category"`
	MediaGallery        []MediaGallery    `json:"media_gallery"`
	Stock               []Stock           `json:"stock"`
	// OPEN: configurable_options
}

type ProductMap struct {
	products map[uint64]*Product
	images   *ImageMap
}

func NewProductMap() (pm *ProductMap, err error) {
	pm = new(ProductMap)
	pm.products = make(map[uint64]*Product)
	pm.images, err = NewImageMap()
	if err != nil {
		return pm, err
	}

	return pm, nil
}

func (pm *ProductMap) Add(f *feed.Product, attr *AttributeMap, cat *CategoryMap) (err error) {
	var (
		p *Product
	)
	dest, err := GetDestination(f)
	if err != nil {
		return err
	}
	img, err := pm.images.Get(f.ImageURL)
	if err != nil {
		return err
	}

	p = &Product{
		ID:    f.GetKey(),
		Name:  f.Name,
		Image: img,
		SKU:   f.SKU,
		//URLKey:
		URLPath:      dest.Link,
		TypeID:       "simple",
		Price:        dest.RegularPrice,
		SpecialPrice: dest.SalesPrice,
		//PriceInclTax:
		//SpecialPriceInclTax:
		//SpecialToDate:
		//SpecialFromDate:
		Status:     1,
		Visibility: 4,
		//Size:
		//Color:
		SizeOptions:  attr.GetIDs(dest.Sizes...),
		ColorOptions: attr.GetIDs(f.ColorGroups...),
		//CategoryIds
	}
	if p.ID == 0 {
		return fmt.Errorf("No ID created for p.Name")
	}

	pm.products[p.ID] = p

	return nil
}

// Dump returns the contents of a product dump
func (m *ProductMap) Dump() (dump []byte, err error) {
	dump, err = json.Marshal(m.products)
	if err != nil {
		return dump, err
	}
	if dump == nil {
		return dump, fmt.Errorf("Product Map is empty")
	}
	return dump, nil
}
