package wooclient

// Attribute provides additional general fields for the products
type Attribute struct {
	ID      int32    `json:"id"`
	Name    string   `json:"name,omitempty"`
	Option  string   `json:"option,omitempty"`  // "term"
	Options []string `json:"options,omitempty"` // "terms"
	Slug    string   `json:"slug,omitempty"`
	Visible bool     `json:"visible,omitempty"`
	Type    string   `json:"type,omitempty"` // "select" by default
	Locale  string   `json:"lang,omitempty"`
}

// GetID implements Item for Attributes
func (a Attribute) GetID() int32 {
	return a.ID
}
