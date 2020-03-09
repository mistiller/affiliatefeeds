package awin

import (
	"fmt"
	"strconv"
	"strings"

	ac "stillgrove.com/gofeedyourself/pkg/awin/client"
	"stillgrove.com/gofeedyourself/pkg/collection"
	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

var (
	// Locales maps tradedoubler language names to WP locales
	Locales = map[string]string{
		"sv":    "sv_se",
		"en":    "sv_se", // FOR TESTING ONLY !
		"sv_se": "sv_se",
	}
	FemaleTerms = [...]string{
		"women",
		"woman",
		"female",
	}
	MaleTerms = [...]string{
		"men",
		"man",
		"male",
	}
	UnisexTerms = [...]string{
		"unisex",
	}
)

type Product struct {
	*ac.Product
	mapping *feed.Mapping
	locale  string
}

// ToFeedProduct returns pointer to a converted FeedProduct
func (p *Product) ToFeedProduct() (productOut *feed.Product, err error) {
	productOut = &feed.Product{
		Name:  p.ProductName,
		SKU:   collection.CollateStrings(p.EAN, p.GTIN, p.MerchantProductID, p.AWProductID, p.ISBN),
		Color: collection.CollateStrings(p.Colour, "multi"),
		//ShortDescription: p.ProductShortDescription,
		Description: collection.CollateStrings(p.Description, p.ProductShortDescription, p.PromotionalText),
		ImageURL:    collection.CollateStrings(p.MerchantImageURL, p.AWImageURL),
		Brand:       p.BrandName,
		Language:    handleLanguage(p.Language, p.locale),
		Retailers: []feed.Retailer{
			feed.Retailer{
				Link: p.MerchantDeepLink,
				Name: p.MerchantName,
				Price: collection.CollateStrings(
					p.SearchPrice,
					p.BasePrice,
					p.BasePriceAmount,
					p.BasePriceText,
				),
				Currency:     p.Currency,
				Availability: handleAvailability(p.InStock, p.StockQuantity, p.StockStatus),
				IsCrawler:    false,
				Sizes:        handleSizes(p.Size),
			},
		},
		ExpectedValue: float32(p.ExpectedValue),
	}

	feedID, _ := strconv.Atoi(p.DataFeedID)
	if feedID == 0 {
		feedID = 999
	}
	productOut.FromFeeds = []int32{int32(feedID)}

	// ----------------------
	// Price logic ----------
	// ----------------------

	lowPrice, highPrice, nPrices := p.GetPrices()
	if nPrices == 0.0 {
		return productOut, fmt.Errorf("Failed to parse prices")
	}
	productOut.HighestPrice, productOut.LowestPrice = float32(lowPrice), float32(highPrice)
	err = productOut.CalculateDiscounts(10)
	if err != nil {
		return productOut, fmt.Errorf("Failed to parse prices - %v", err)
	}

	// ----------------------
	// Color logic ----------
	// ----------------------

	colorCandidates := []string{
		p.Colour,
		productOut.Color,
		strings.Replace(p.ProductName, p.BrandName, "", 1),
	}
	productOut.ColorGroups = handleColors(
		p.mapping,
		colorCandidates...,
	)
	if len(productOut.ColorGroups) == 0 {
		return productOut, fmt.Errorf("Failed to parse color - %v", colorCandidates)
	}

	// ----------------------
	// Gender logic ---------
	// ----------------------

	productOut.Gender = handleGender(
		p.MerchantCategory,
		p.CategoryName,
		p.Custom1,
		p.Custom2,
	)
	if productOut.Gender == "" {
		return productOut, fmt.Errorf("Failed to parse gender - %v", p.MerchantCategory)
	}

	// ----------------------
	// Category logic -------
	// ----------------------

	productOut.ProviderCategories, err = handleCategories(
		p.mapping,
		p.MerchantCategory,
		p.CategoryName,
		p.Custom1,
		p.Custom2,
	)

	productOut.OriginalCategories = []string{p.MerchantCategory}

	if p.MerchantCategory == "" && err != nil {
		return productOut, fmt.Errorf("Failed to parse categories - %v", err)
	}

	// ----------------------
	// Offer logic ----------
	// ----------------------

	productOut.RetailerMap = make(map[uint64]struct{}, 1)
	productOut.RetailerMap[collection.HashKey(p.MerchantDeepLink)] = struct{}{}

	for i := range productOut.Retailers {
		if productOut.Retailers[i].Availability == "instock" {
			productOut.Active = true
		}
	}

	// ----------------------
	// SKU logic ------------
	// ----------------------

	productOut.SKU = collection.CollateStrings(
		p.AWProductID,
		p.MerchantProductID,
		p.EAN,
		p.ISBN,
	)

	if productOut.SKU == "" {
		return productOut, fmt.Errorf("No identifier found - %v", p)
	}

	err = productOut.SetKey()
	if err != nil {
		return productOut, fmt.Errorf("Failed to set key - %v", err)
	}

	return productOut, nil
}

func handleSizes(str ...string) (out []string) {
	var (
		tmp []string
	)
	for i := range str {
		tmp = strings.Split(str[i], ",")
		for j := range tmp {
			out = append(
				out,
				strings.TrimSpace(tmp[j]),
			)
		}
	}
	return out
}

func handleAvailability(str ...string) string {
	for i := range str {
		switch str[i] {
		case "1", "instock", "in_stock", "in stock", "true", "yes":
			return "instock"
		default:
			return "out of stock"
		}
	}
	return "out of stock"
}

func handleColors(mapping *feed.Mapping, str ...string) (colors []string) {
	var (
		substr []string
		terms  []string
	)

	for s := range str {
		substr = collection.SplitList(str[s])
		for i := range substr {
			terms = append(terms, collection.Sanitize(substr[i]))
		}

		if strings.ToLower(str[s]) == "no color" {
			colors = append(colors, "multi")
		}
	}

	for i := range terms {
		replacements, matched := collection.StrictFindReplace2(terms[i], mapping.ColorMap)
		if !matched {
			colors = append(
				colors,
				terms[i],
			)
		}
		for j := range replacements {
			colors = append(
				colors,
				replacements[j],
			)
		}
	}

	return collection.UniqueNames(colors)
}

func handleGender(str ...string) string {
	var (
		substr []string
	)
	for s := range str {
		substr = collection.SplitList(str[s])
		for i := range substr {
			for j := range FemaleTerms {
				if strings.ContainsAny(strings.ToLower(substr[i]), FemaleTerms[j]) {
					return "women"
				}
			}
			for j := range MaleTerms {
				if strings.ContainsAny(strings.ToLower(substr[i]), MaleTerms[j]) {
					return "men"
				}
			}
			for j := range UnisexTerms {
				if strings.ContainsAny(strings.ToLower(substr[i]), UnisexTerms[j]) {
					return "unisex"
				}
			}
		}
	}

	return ""
}

func handleCategories(mapping *feed.Mapping, str ...string) (categories []feed.ProviderCategory, err error) {
	var (
		substr []string
		terms  []string
		gender rune
		exists bool
	)

	uniques := make(map[string]struct{})
	for s := range str {
		substr = collection.SplitList(str[s])
		for i := range substr {
			term := strings.ToLower(substr[i])
			_, exists = uniques[term]
			if exists {
				continue
			}
			terms = append(terms, term)
			uniques[term] = struct{}{}
		}
	}

	switch handleGender(terms...) {
	case "women":
		gender = 'w'
		break
	case "unisex":
		gender = 'u'
		break
	case "men":
		gender = 'm'
		break
	default:
		return categories, fmt.Errorf("Couldn't parse gender - %v", terms)
	}

	for i := range terms {
		replacements, matched := collection.StrictFindReplace2(terms[i], mapping.CatNameMap)
		if !matched {
			continue
		}
		for j := range replacements {
			categories = append(
				categories,
				feed.ProviderCategory{
					ProviderName: "awin",
					Name:         strings.TrimSpace(replacements[j]),
					Gender:       gender,
				},
			)
		}
	}

	if len(categories) == 0 {
		return categories, fmt.Errorf("No categories found for - %v", terms)
	}

	return categories, nil
}

func handleLanguage(str ...string) string {
	var (
		exists bool
	)
	for i := range str {
		_, exists = Locales[str[i]]
		if exists == true {
			return Locales[str[i]]
		}
	}
	return str[0]
}
