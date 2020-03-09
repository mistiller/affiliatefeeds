package tradedoublerclient

// Price is the object of the PriceHistory
type Price struct {
	Date  int64             `json:"date"`
	Price map[string]string `json:"price"`
}

// Offer contains the shops
type Offer struct {
	FeedID          int32   `json:"feedId"`
	ProductURL      string  `json:"productUrl"`
	Modified        int64   `json:"modified"`
	SourceProductID string  `json:"sourceProductId"`
	ProgramLogo     string  `json:"programLogo"`
	ProgramName     string  `json:"programName"`
	Availability    string  `json:"availability"`
	Condition       string  `json:"condition"`
	PriceHistory    []Price `json:"priceHistory"`
	ShippingCost    string  `json:"shippingCost"`
	DeliveryTime    string  `json:"deliveryTime"`
}

// Image contains the image resources as given in the feed
type Image struct {
	URL    string `json:"url"`
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
}

// Product contains the fields that TradeDoubler supplies for each product
type Product struct {
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	ShortDescription string                   `json:"shortDescription"`
	Brand            string                   `json:"brand"`
	Categories       []Category               `json:"categories"`
	Fields           []map[string]interface{} `json:"fields"`
	Language         string                   `json:"language"`
	Identifiers      map[string]string        `json:"identifiers"`
	ProductImage     Image                    `json:"productImage"`
	Offers           []Offer                  `json:"offers"`
	ID               int32                    `json:"id"`
	FeedID           int32                    `json:"feedId"`
	ProgramName      string                   `json:"programName"`
	SKU              string                   `json:"sku"`
	GTIN             string                   `json:"gtin"`
	ItemGroupID      string                   `json:"item_group_id"`
}
