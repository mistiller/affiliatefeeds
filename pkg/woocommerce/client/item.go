package wooclient

// Item is the object used to hold requests and responses towards the woocommerce api
// Examples: Product, Attribute, Category
type Item interface {
	GetID() int32
}
