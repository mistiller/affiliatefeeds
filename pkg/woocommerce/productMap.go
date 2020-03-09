package woocommerce

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// ProductMap stores a map of product pointers and offers convenience functions
type ProductMap struct {
	products    map[uint64]*Product
	nProducts   uint64
	nCategories uint64
	initialized bool
}

func PMFromPM(f *feed.ProductMap, mappings *ProductMapping) (pm *ProductMap, err error) {
	var (
		exist   bool
		lastErr error
		id      uint64
		failed  uint64
	)
	pm = new(ProductMap)
	pm.products = make(map[uint64]*Product)

	categories := make(map[int32]struct{})

	inProducts, nProducts, nFeeds, nCategories := f.Get()
	if nProducts*nFeeds*nCategories == 0 {
		return pm, fmt.Errorf("PMFromPM - Product Map Inconsistent")
	}

	temp := new(FeedProduct)
	wp := new(Product)
	for k := range inProducts {
		temp = &FeedProduct{
			*inProducts[k],
		}
		wp, err = temp.ToWooProduct(mappings)
		if err != nil {
			log.WithFields(
				log.Fields{
					"Name":      temp.Name,
					"Operation": "Feed Product To Woo Product",
					"Error":     err,
				},
			).Debugln("Dropped Product")
			failed++
			lastErr = err
			continue
		}
		id = uint64(wp.GetKey())
		if id == 0 {
			log.WithFields(
				log.Fields{
					"Name":      temp.Name,
					"Operation": "Feed Product To Woo Product",
					"Error":     "No valid product ID",
				},
			).Debugln("Dropped Product")
			failed++
			continue
		}
		_, exist = pm.products[id]
		if !exist {
			pm.products[id] = wp
			for i := range wp.Categories {
				_, exist = categories[wp.Categories[i].ID]
				if !exist {
					categories[wp.Categories[i].ID] = struct{}{}
				}
			}
		}
	}

	pm.nProducts = uint64(len(pm.products))
	pm.nCategories = uint64(len(categories))

	if pm.nProducts == 0 {
		return pm, fmt.Errorf("Failed to convert any products - %v", lastErr)
	}

	log.WithFields(
		log.Fields{
			"Products Filtered Out": failed,
			"Products Converted":    fmt.Sprintf("%d / %d", pm.nProducts, nProducts),
			"Categories Mapped":     fmt.Sprintf("%d / %d", pm.nCategories, nCategories),
		},
	).Infoln("Converted Product Feed To WooCommerce Format")

	pm.initialized = true
	return pm, nil
}

// Get returns map of product pointers, number of products, and number of categories
func (pm *ProductMap) Get() (products map[uint64]*Product, nProducts, nCategories uint64) {
	if !pm.initialized {
		log.Errorln("Product Map not initialized")
		return pm.products, 0, 0
	}
	return pm.products, pm.nProducts, pm.nCategories
}

func (pm *ProductMap) GetCategories() (categories []int32) {
	var exist bool
	cats := make(map[int32]struct{})
	for i := range pm.products {
		for j := range pm.products[i].Categories {
			_, exist = cats[pm.products[i].Categories[j].ID]
			if !exist {
				cats[pm.products[i].Categories[j].ID] = struct{}{}
				categories = append(categories, pm.products[i].Categories[j].ID)
			}
		}
	}
	return categories
}

// GetGroups splits the product map into create, update, and delete maps to be sent to the API
func (pm *ProductMap) GetGroups(oldProducts map[uint64]uint64) (create map[uint64]*Product, update map[uint64]*Product, delete []int, err error) {
	return getGroups(pm.products, oldProducts)
}

func (pm *ProductMap) Flush() {
	pm.products = nil
}

func getGroups(newProducts map[uint64]*Product, oldProducts map[uint64]uint64) (create, update map[uint64]*Product, delete []int, err error) {
	var (
		exist bool
	)

	touched := make(map[uint64]struct{})
	create = make(map[uint64]*Product)
	update = make(map[uint64]*Product)

	for k := range oldProducts {
		if k == 0 {
			continue
		}
		_, exist = newProducts[k]
		if !exist {
			delete = append(delete, int(oldProducts[k]))
		} else {
			np := *newProducts[k]
			update[k] = &np
			update[k].Name = ""
			update[k].ID = oldProducts[k]
		}
		touched[k] = struct{}{}
	}

	for k2 := range newProducts {
		if k2 == 0 {
			continue
		}
		if newProducts[k2].GetKey() == 0 {
			continue
		}
		_, exist = touched[k2]
		if exist {
			continue
		}

		_, exist = create[k2]
		if !exist {
			np := *newProducts[k2]
			create[k2] = &np
			create[k2].ID = 0
			for i := range create[k2].Images {
				create[k2].Images[i].ID = 0
			}
		}
	}

	if len(create)+len(update)+len(delete) == 0 {
		return create, update, delete, fmt.Errorf("No updates prepared")
	}

	return create, update, delete, nil
}
