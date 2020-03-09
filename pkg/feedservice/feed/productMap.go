package feed

import (
	"encoding/csv"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// ProductMap stores a unique set of feed products
type ProductMap struct {
	products    map[uint64]*Product
	nProducts   uint64
	nFeeds      uint64
	nCategories uint64
	nBrands     uint64
	nRetailers  uint64
	validated   bool
	retailers   map[string]struct{}
	feeds       map[int32]struct{}
}

// PMFromSlice generates ProductMap from Product slice
func PMFromSlice(products []Product) (m *ProductMap, err error) {
	var (
		exist   bool
		key     uint64
		lastErr error
	)

	if len(products) == 0 {
		return m, fmt.Errorf("Slice To PM - No products in slice")
	}

	m = new(ProductMap)
	m.products = make(map[uint64]*Product, len(products))

	for idx := range products {
		err = products[idx].Update()
		if err != nil {
			lastErr = err
			log.Println(err)
			continue
		}

		key = products[idx].GetKey()
		_, exist = m.products[key]
		if !exist {
			m.products[key] = &products[idx]
			m.nProducts++

		} else {
			err = m.products[key].MergeWith(&products[idx])
			if err != nil {
				return m, err
			}
		}
	}

	err = m.eval()
	if err != nil {
		return m, fmt.Errorf("Validate new Product Map - %v", err)
	}

	if m.nProducts == 0 {
		return m, fmt.Errorf("Slice to PMap - products: %d, feeds: %d, categories: %d - %v", m.nProducts, m.nFeeds, m.nCategories, lastErr)
	}

	return m, nil
}

// PMFromMap generates ProductMap from Product slice
func PMFromMap(products map[uint64]*Product) (m *ProductMap, err error) {
	var (
		exist   bool
		lastErr error
		key     uint64
	)

	m = new(ProductMap)
	m.products = make(map[uint64]*Product, len(products))
	for k := range products {
		err = products[k].Update()
		if err != nil {
			lastErr = err
			log.WithFields(
				log.Fields{
					"ID":    k,
					"Name":  products[k].Name,
					"Error": err,
				},
			).Debugln("Dropping Product")
			continue
		}

		key = products[k].GetKey()
		_, exist = m.products[key]
		if !exist {
			m.products[key] = products[k]
			m.nProducts++
		} else {
			err = m.products[key].MergeWith(products[k])
			if err != nil {
				return m, err
			}
		}
		m.nProducts++
	}
	err = m.eval()
	if err != nil {
		return m, fmt.Errorf("PMFromMap - eval - %v", err)
	}
	if m.nProducts*m.nFeeds*m.nCategories == 0 {
		return m, fmt.Errorf("Map to PMap - Product List incomplete - products: %d, feeds: %d, categories: %d - %v", m.nProducts, m.nFeeds, m.nCategories, lastErr)
	}

	return m, nil
}

// Get returns map of unique feed products
func (m *ProductMap) Get() (products map[uint64]*Product, nProducts uint64, nFeeds uint64, nCategories uint64) {
	if !m.validated {
		err := m.eval()
		if err != nil {
			return m.products, m.nProducts, m.nFeeds, m.nCategories
		}
	}
	return m.products, m.nProducts, m.nFeeds, m.nCategories
}

// DumpToCSV writes product map into a csv file
func (m *ProductMap) DumpToCSV(filename string) (err error) {
	var data [][]string
	if !m.validated {
		err := m.eval()
		if err != nil {
			return err
		}
	}

	var (
		header []string
		rows   [][]string
	)
	for k := range m.products {
		header, rows, err = m.products[k].AsRows()
		if err != nil {
			//log.Debugf("%v - %s", err, m.products[k].Name)
			continue
		}
		for i := range rows {
			data = append(data, rows[i])
		}
	}

	f, err := os.Create(filename)
	writer := csv.NewWriter(f)
	if err != nil {
		return err
	}
	defer writer.Flush()

	err = writer.Write(header)
	if err != nil {
		return err
	}

	for i := range data {
		err := writer.Write(data[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// Stats returns number of products, feeds, and categories in Product Map
func (m *ProductMap) Stats() (nProducts uint64, nFeeds uint64, nCategories uint64) {
	if !m.validated {
		err := m.eval()
		if err != nil {
			return m.nProducts, m.nFeeds, m.nCategories
		}
	}
	return m.nProducts, m.nFeeds, m.nCategories
}

// GetBrands returns a slice of the unique brand names in the map
func (m *ProductMap) GetBrands() (brands []string) {
	var (
		exist bool
		count uint64
	)
	if !m.validated {
		err := m.eval()
		if err != nil {
			return brands
		}
	}
	bm := make(map[string]struct{})
	for k := range m.products {
		_, exist = bm[m.products[k].Brand]
		if !exist {
			bm[m.products[k].Brand] = struct{}{}
			brands = append(brands, m.products[k].Brand)
			count++
		}
		if count >= m.nBrands {
			break
		}
	}
	return brands
}

func (m *ProductMap) GetFeeds() (feeds []int32) {
	if len(m.feeds) == 0 {
		err := m.eval()
		if err != nil {
			return feeds
		}
	}

	feeds = make([]int32, len(m.feeds))
	i := 0
	for k := range m.feeds {
		feeds[i] = k
		i++
	}

	return feeds
}

func (m *ProductMap) GetRetailers() (retailers []string) {
	if len(m.retailers) == 0 {
		err := m.eval()
		if err != nil {
			return retailers
		}
	}

	retailers = make([]string, len(m.retailers))
	i := 0
	for k := range m.retailers {
		retailers[i] = k
		i++
	}
	return retailers
}

func (m *ProductMap) eval() (err error) {
	var (
		exist bool
		//lastErr error
	)

	m.feeds = make(map[int32]struct{})
	m.retailers = make(map[string]struct{})
	categories := make(map[string]struct{})
	brands := make(map[string]struct{})

	for k := range m.products {
		if !m.products[k].Active {
			delete(m.products, k)
			continue
		}
		err = m.products[k].Update()
		if err != nil {
			//lastErr = err
			log.WithFields(
				log.Fields{
					"ID":    k,
					"Name":  m.products[k].Name,
					"Error": err,
				},
			).Debugln("Dropping Product")
			delete(m.products, k)
			continue
		}
		err = m.products[k].Validate()
		if err != nil {
			log.WithFields(
				log.Fields{
					"ID":    k,
					"Name":  m.products[k].Name,
					"Error": err,
				},
			).Warningln("Dropping Product")
			delete(m.products, k)
			continue
		}
		for i := range m.products[k].FromFeeds {
			_, exist = m.feeds[m.products[k].FromFeeds[i]]
			if !exist {
				m.feeds[m.products[k].FromFeeds[i]] = struct{}{}
				m.nFeeds++
			}
		}
		for j := range m.products[k].ProviderCategories {
			_, exist = categories[m.products[k].ProviderCategories[j].Name]
			if !exist {
				categories[m.products[k].ProviderCategories[j].Name] = struct{}{}
				m.nCategories++
			}
		}
		for key := range m.products[k].Retailers {
			_, exist = m.retailers[m.products[k].Retailers[key].Name]
			if !exist {
				m.retailers[m.products[k].Retailers[key].Name] = struct{}{}
				m.nRetailers++
			}
		}
		_, exist = brands[m.products[k].Brand]
		if !exist {
			brands[m.products[k].Brand] = struct{}{}
			m.nBrands++
		}
	}
	m.validated = true
	return nil
}
