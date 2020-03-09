package feed

// Feed is implemented via a get method that generates an array of products
type Feed interface {
	GetName() string
	Get(productionFlag bool) ([]Product, error)
	GetLocale() *Locale
}
