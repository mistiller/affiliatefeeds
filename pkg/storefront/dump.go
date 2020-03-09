package storefront

import (
	"fmt"
	"io/ioutil"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// Dump contains attributes, categories, and products to be written to a vsf-compatible JSON file
type Dump struct {
	Products   *ProductMap
	Attributes *AttributeMap
	Categories *CategoryMap
}

func NewFromFeed(pm *feed.ProductMap) (d *Dump, err error) {
	c := NewCounter()

	attributes := NewAttributeMap(c)
	categories := NewCategoryMap(c)

	products, err := NewProductMap()
	if err != nil {
		return d, err
	}

	feed, _, _, _ := pm.Get()
	for i := range feed {
		attributes.Add("Gender", feed[i].Gender)
		attributes.Add("Brand", feed[i].Brand)

		attributes.Add("Color", feed[i].Color)
		for j := range feed[i].ColorGroups {
			attributes.Add("Color", feed[i].ColorGroups[j])
		}

		for j := range feed[i].DiscountBins {
			attributes.Add("DiscountLevel", feed[i].DiscountBins[j])
		}

		err = categories.Add(feed[i].ProviderCategories)
		if err != nil {
			return d, err
		}
	}

	for i := range feed {
		err = products.Add(feed[i], attributes, categories)
		if err != nil {
			return d, err
		}
	}
	d = &Dump{
		Attributes: attributes,
		Categories: categories,
		Products:   products,
	}

	return d, nil
}

func (d *Dump) GetData() (attributes, categories, products []byte, err error) {
	attributes, err = d.Attributes.Dump()
	if err != nil {
		return attributes, categories, products, fmt.Errorf("Failed to create attribute dump - %v", err)
	}
	categories, err = d.Categories.Dump()
	if err != nil {
		return attributes, categories, products, fmt.Errorf("Failed to create category dump - %v", err)
	}
	products, err = d.Products.Dump()
	if err != nil {
		return attributes, categories, products, fmt.Errorf("Failed to create product dump - %v", err)
	}

	return attributes, categories, products, nil
}

// WriteFiles - Write products feed, attributes, and category maps to 3 json files in filepath
// "attributes.json", "categories.json", "products.json"
func (d *Dump) WriteFiles(filepath string) (err error) {
	type file []byte
	var files [3]file
	files[0], files[1], files[2], err = d.GetData()
	if err != nil {
		return fmt.Errorf("Failed to write product dump - %v", err)
	}

	names := []string{"attributes.json", "categories.json", "products.json"}
	for i := 0; i < len(files); i++ {
		err = ioutil.WriteFile(
			filepath+"/"+names[i],
			files[i],
			0644,
		)
		if err != nil {
			return fmt.Errorf("Failed to write product dump - %s - %v", names[i], err)
		}
	}

	return nil
}
