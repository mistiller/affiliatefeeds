package storefront

import (
	"encoding/json"
	"fmt"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

type Category struct {
	ID           uint64        `json:"id"`
	ParentID     uint64        `json:"parent_id"`
	Name         string        `json:"name"`
	ParentName   string        `json:"-"`
	URLKey       string        `json:"url_key"`
	Path         string        `json:"path"`
	URLPath      string        `json:"url_path"`
	IsActive     bool          `json:"is_active"`
	Position     int           `json:"position"`
	Level        int           `json:"level"`
	ProductCount int           `json:"product_count"`
	ChildrenData []interface{} `json:"children_data"`
}

type CategoryMap struct {
	categories map[uint64]*Category
	nitems     int
	lookup     map[string]uint64
	counter    *IDCounter
}

func NewCategoryMap(counter *IDCounter) *CategoryMap {
	if counter.Get() == 0 {
		counter.Increment(1)
	}

	return &CategoryMap{
		categories: make(map[uint64]*Category),
		counter:    counter,
		lookup:     make(map[string]uint64),
	}
}

// IMPORTANT: Fix Category Tree Logic !!!
// ###
// Add take in parent: child maps of category names
// and sorts them in such a way that we get
func (m *CategoryMap) Add(categories []feed.ProviderCategory) (err error) {
	var (
		exists bool
		// name: parentName
	)

	parents := make(map[string][]string)
	children := make(map[string][]string)

	for i := range categories {
		parent := string(categories[i].Gender)

		_, exists = parents[parent]
		if !exists {
			parents[parent] = []string{""}
		}
	}
	for i := range categories {
		parent := string(categories[i].Gender)

		_, exists = parents[categories[i].Name]
		if exists {
			parents[categories[i].Name] = append(
				parents[categories[i].Name],
				parent,
			)
		}

		_, exists = children[categories[i].Name]
		if !exists {
			children[categories[i].Name] = append(
				children[categories[i].Name],
				parent,
			)
		}
	}

	return nil
}

// add loops through existing categories
// and adds unique name / parent combinations
func (m *CategoryMap) add(name, parent string) (done bool) {
	var (
		parentID uint64
	)
	for k := range m.categories {
		if m.categories[k].Name == name && m.categories[k].ParentName == parent {
			return true
		}
	}
	for _, val := range m.GetIDs(parent) {
		parentID = val
	}
	if parentID == 0 {
		return false
	}
	id := m.counter.Get() + 1
	m.categories[id] = &Category{
		ID:       id,
		ParentID: parentID,
		Name:     name,
		IsActive: true,
	}
	m.counter.Increment(1)
	//m.nitems++
	return true
}

func (m *CategoryMap) GetIDs(str ...string) (out []uint64) {
	var (
		val    uint64
		exists bool
	)

	if m.lookup == nil {
		return out
	}

	for i := range str {
		val, exists = m.lookup[str[i]]
		if exists {
			out = append(out, val)
		}
	}

	return out
}

// Dump returns the contents of a category dump
func (m *CategoryMap) Dump() (dump []byte, err error) {
	dump, err = json.Marshal(m.categories)
	if err != nil {
		return dump, err
	}
	if dump == nil {
		return dump, fmt.Errorf("Category Map is empty")
	}
	return dump, nil
}
