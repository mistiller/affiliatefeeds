package wooclient

// Brand is an object provided by the perfect woocommerce brand plugin
type Brand struct {
	TermID         int32  `json:"term_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Slug           string `json:"slug,omitempty"`
	TermGroup      int32  `json:"term_group,omitempty"`
	TermTaxonomyID int32  `json:"term_taxonomy_id,omitempty"`
	Taxonomy       string `json:"taxonomy,omitempty"`
	Description    string `json:"description,omitempty"`
	Parent         int32  `json:"parent,omitempty"`
	Count          int32  `json:"count,omitempty"`
	Filter         string `json:"filter,omitempty"`
	TermOrder      string `json:"term_order,omitempty"`
	BrandImage     bool   `json:"brand_image,omitempty"`
	BrandBanner    bool   `json:"brand_banner,omitempty"`
	Locale         string `json:"lang,omitempty"`
}

// GetID implements Item for Brands
func (b Brand) GetID() int32 {
	return b.TermID
}
