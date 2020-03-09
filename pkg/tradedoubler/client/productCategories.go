package tradedoublerclient

// Category can be filled recursively and cover any level lower than 2
type Category struct {
	iterator      int
	Name          string     `json:"name,omitempty"`
	ID            int        `json:"id,omitempty"`
	ProductCount  int        `json:"productCount,omitempty"`
	SubCategories []Category `json:"subCategories,omitempty"`
	CategoryName  string     `json:"tdCategoryName,omitempty"`
}

// CategoryTree is level 1 of the tree
type CategoryTree struct {
	iterator      int
	Language      string     `json:"language,omitempty"`
	Name          string     `json:"name,omitempty"`
	ID            int        `json:"id,omitempty"`
	ProductCount  int        `json:"productCount,omitempty"`
	SubCategories []Category `json:"subCategories,omitempty"`
}

// ProductCategories contains a list of Category Trees
type ProductCategories struct {
	CategoryTrees []CategoryTree `json:"categoryTrees"`
}

// GetTree iteratets over the CategoryTrees, returns nil when finished
func (pc *ProductCategories) GetTree() CategoryTree {
	return pc.CategoryTrees[0]
}

// NextSubCategory iterates over the SubCategories of a category, returns nil when finished
func (ct *CategoryTree) NextSubCategory() *Category {
	if len(ct.SubCategories) <= ct.iterator+1 {
		return nil
	}
	out := &ct.SubCategories[ct.iterator]
	ct.iterator++

	return out
}

// NextSubCategory iterates over the SubCategories of a category, returns nil when finished
func (c *Category) NextSubCategory() *Category {
	if len(c.SubCategories) <= c.iterator+1 {
		return nil
	}
	out := &c.SubCategories[c.iterator]
	c.iterator++

	return out
}
