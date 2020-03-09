package woocommerce

import (
	"fmt"
	"strings"

	"stillgrove.com/gofeedyourself/pkg/collection"
	c "stillgrove.com/gofeedyourself/pkg/collection"
	gwc "stillgrove.com/gofeedyourself/pkg/woocommerce/client"
)

// Product inherits for gwc.Product but adds specific validation method
type Product struct {
	gwc.Product
	ActiveStore  string   `json:"-"`
	LowestPrice  float32  `json:"-"`
	HighestPrice float32  `json:"-"`
	Sizes        []string `json:"-"`
}

// Validate returns error or nil depending whether the product fullfills our standards of consistency and completeness
func (p *Product) Validate(attributes []string) (err error) {
	if c.AnyEmpty(
		[]*string{
			&p.Name,
			&p.SKU,
			&p.Lang,
			&p.Type,
			&p.RegularPrice,
			&p.ButtonText,
		}) {
		return fmt.Errorf("Validation - Every product needs Name and SKU and Language")
	}
	desc := c.CollateString(p.Description, p.ShortDescription)
	if c.IsEmpty(&desc) {
		return fmt.Errorf("Validation - Missing description field")
	}

	if c.IsEmpty(&p.RegularPrice) {
		return fmt.Errorf("Validation - Prices are inconsistent")
	}
	for i := range p.Attributes {
		if p.Attributes[i].Name == "Discount Level" && c.IsEmpty(&p.SalePrice) {
			return fmt.Errorf("Validation - Prices are inconsistent")
		}

		if p.Attributes[i].Name == "Brand" {
			if len(p.Attributes[i].Options) != 1 {
				return fmt.Errorf("Only exactly 1 brand option allowed")
			}
		}
	}

	if c.IsEmpty(&p.ExternalURL) ||
		strings.Contains(p.ExternalURL, "stillgrove") {
		return fmt.Errorf("Missing valid external URL - %s - %s", p.Name, p.ExternalURL)
	}

	if p.Lang != p.Language {
		return fmt.Errorf("Languages inconsistent: %s - %s", p.Lang, p.Language)
	}

	err = p.validateAttributes(attributes)
	if err != nil {
		return fmt.Errorf("Validation - Attributes - %v", err)
	}

	err = p.validateCategories()
	if err != nil {
		return fmt.Errorf("Validation - Categories - %v", err)
	}

	if len(p.Images) == 0 {
		return fmt.Errorf("Validation - No images assigned")
	}

	var matched bool
	for i := range p.Images {
		if c.IsEmpty(&p.Images[i].Name) || c.IsEmpty(&p.Images[i].SRC) {
			return fmt.Errorf("Validation - Image invalid - %s", p.Categories[i].Name)
		}

		s := "y"
		var formatMap = map[string]*string{
			".png":  &s,
			".jpg":  &s,
			".jpeg": &s,
			".gif":  &s,
		}
		_, matched = collection.StrictFindReplace(
			p.Images[i].SRC,
			formatMap,
		)
		if !matched {
			return fmt.Errorf("Validation - Wrong Image type - %s", p.Images[i].SRC)
		}
	}

	return nil
}

func (p *Product) validateAttributes(toFind []string) (err error) {
	var (
		uniques  map[string]struct{}
		checkMap map[string]struct{}
		exists   bool
	)
	if len(p.Attributes) == 0 {
		return fmt.Errorf("No attributes assigned")
	}

	checkMap = make(map[string]struct{})

	for i := range p.Attributes {
		if p.Attributes[i].GetID() == 0 {
			return fmt.Errorf("Attribute invalid - %s", p.Attributes[i].Name)
		}

		if p.Attributes[i].Option == p.Name {
			return fmt.Errorf("Attribute has product name - %s", p.Attributes[i].Name)
		}
		for j := range p.Attributes[i].Options {
			uniques = make(map[string]struct{})
			if p.Attributes[i].Options[j] == p.Name {
				return fmt.Errorf("Attribute has product name - %s", p.Attributes[i].Name)
			}
			_, exists = uniques[p.Attributes[i].Options[j]]
			if exists {
				return fmt.Errorf("Attribute has duplicate option - %v", p.Attributes[i].Options)
			}
		}

		for j := range toFind {
			if p.Attributes[i].Name == toFind[j] {
				checkMap[toFind[j]] = struct{}{}
				break
			}
		}
	}
	if len(checkMap) < len(toFind) {
		var exists bool
		for k := range toFind {
			if c.IsEmpty(&toFind[k]) {
				continue
			}
			_, exists = checkMap[toFind[k]]
			if exists == false {
				return fmt.Errorf("Missing attribute field - %s", toFind[k])
			}
		}
	}

	return nil
}

func (p *Product) validateCategories() (err error) {
	var key int32
	var exist bool

	if AllowMultiCats == true {
		if len(p.Categories) < 2 {
			return fmt.Errorf("Validation - Fewer than 2 categories assigned")
		}
	}

	cats := make(map[int32]struct{})
	for i := range p.Categories {
		if p.Categories[i].GetID() == 0 {
			return fmt.Errorf("Validation - Category invalid - %s", p.Categories[i].Name)
		}

		key = p.Categories[i].ID
		_, exist = cats[key]
		if exist {
			return fmt.Errorf("Category IDs not unique - %v", p.Categories)
		}
		cats[key] = struct{}{}
	}

	return nil
}
