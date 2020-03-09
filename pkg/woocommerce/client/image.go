package wooclient

/*---------------------------------------------------------------------
Can image dupliaction be avoided by managing the IDs akin to brands? --
----------------------------------------------------------------------*/

// Image contains all the information on product images
type Image struct {
	ID int32 `json:"id,omitempty"` // Read Only ???
	//DateCreated     string `json:"date_created,omitempty"`
	DateCreatedGMT string `json:"date_created_gmt,omitempty"`
	//DateModified    string `json:"date_modified,omitempty"`
	DateModifiedGMT string `json:"date_modified_gmt,omitempty"`
	SRC             string `json:"src,omitempty"`
	Name            string `json:"name,omitempty"`
	Alt             string `json:"alt,omitempty"`
}

//GetID implements Item for Product Images
func (i Image) GetID() int32 {
	return i.ID
}
