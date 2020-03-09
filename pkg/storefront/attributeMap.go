package storefront

import (
	"encoding/json"
	"fmt"

	"stillgrove.com/gofeedyourself/pkg/collection"
)

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Attribute struct {
	ID                   uint64   `json:"id"`
	IsUserDefined        bool     `json:"is_user_defined"`
	IsVisible            bool     `json:"is_visible"`
	FrontendInput        string   `json:"frontend_input"`
	AttributeCode        string   `json:"attribute_code"`
	DefaultValue         string   `json:"default_value"`
	Options              []Option `json:"options"`
	DefaultFrontendLabel string   `json:"default_frontend_label"`
}

type AttributeMap struct {
	attributes map[uint64]*Attribute
	counter    *IDCounter
	lookup     map[string]uint64
	nitems     uint32
}

func NewAttributeMap(counter *IDCounter) *AttributeMap {
	if counter.Get() == 0 {
		counter.Increment(1)
	}

	return &AttributeMap{
		attributes: make(map[uint64]*Attribute),
		lookup:     make(map[string]uint64),
		counter:    counter,
	}
}

// Add loops through existing attributes
// Either adds a new option to an existing attribute, starts a new attrbute on this option, or returns false
func (m *AttributeMap) Add(name, option string) (added bool) {
	id := m.counter.Get() + 1

	for k := range m.attributes {
		if m.attributes[k].DefaultFrontendLabel == name {
			for i := range m.attributes[k].Options {
				if m.attributes[k].Options[i].Label == option {
					return false
				}
			}
			m.attributes[k].Options = append(
				m.attributes[k].Options,
				Option{
					Label: option,
					Value: fmt.Sprintf("%d", id),
				},
			)
			// We just added an option, i.e. one new item
			m.counter.Increment(1)
			//m.nitems++

			return true
		}
	}

	m.attributes[id] = &Attribute{
		ID:            id,
		IsUserDefined: true,
		IsVisible:     true,
		FrontendInput: "select",
		AttributeCode: collection.SanitizeHard(name),
		DefaultValue:  "",
		Options: []Option{
			Option{
				Label: option,
				Value: fmt.Sprintf("%d", id+1),
			},
		},
		DefaultFrontendLabel: name,
	}

	m.lookup[option] = id + 2

	// We just added an attribute and an option, i.e. two new items
	m.counter.Increment(2)
	//m.nitems += 2

	return true
}

func (m *AttributeMap) GetIDs(str ...string) (out []uint64) {
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

// Dump returns the contents of an attribute dump
func (m *AttributeMap) Dump() (dump []byte, err error) {
	dump, err = json.Marshal(m.attributes)
	if err != nil {
		return dump, err
	}
	if dump == nil {
		return dump, fmt.Errorf("Attribute Map is empty")
	}
	return dump, nil
}
