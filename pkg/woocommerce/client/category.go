package wooclient

// CategoryLink can be either self, collection, or up
// e.g.: "https://example.com/wp-json/wc/v3/products/categories/15"
type CategoryLink struct {
	Href string `json:"href,omitempty"`
}

// CategoryLinks are generated from the category ids:
// e.g.: "href": "https://example.com/wp-json/wc/v3/products/categories/15"
type CategoryLinks struct {
	Self       []CategoryLink `json:"self,omitempty"`
	Collection []CategoryLink `json:"collection,omitempty"`
	Up         []CategoryLink `json:"up,omitempty"`
}

// Category convers objects relating to the WC Category tree
type Category struct {
	ID          int32         `json:"id,omitempty"`
	Name        string        `json:"name"`
	Alt         string        `json:"alt,omitempty"`
	Slug        string        `json:"slug,omitempty"`
	Parent      int32         `json:"parent,omitempty"`
	Description string        `json:"description,omitempty"`
	Image       Image         `json:"image,omitempty"`
	MenuOrder   int32         `json:"menu_order,omitempty"`
	Count       int32         `json:"count,omitempty"`
	Links       CategoryLinks `json:"_links,omitempty"` // read-only
}

// GetID implements Item for Catgeories
func (c Category) GetID() int32 {
	return c.ID
}
