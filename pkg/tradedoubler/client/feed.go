package tradedoublerclient

// Feed receives the direct response from the the product search service
// http://api.tradedoubler.com/1.0/products[.xml|.json|empty][query keys]?token={token}[&jsonp=myCallback]
type Feed struct {
	ProductHeader map[string]interface{} `json:"productHeader"`
	Products      []Product              `json:"products"`
}

// Program is an object inside the FeedInfo
type Program struct {
	ProgramID int32  `json:"programId"`
	Name      string `json:"name"`
}

// FeedInfo is the response object for the FeedService
// http://api.tradedoubler.com/1.0/productFeeds?token={token}
type FeedInfo struct {
	FeedID                     uint64    `json:"feedId"`
	Name                       string    `json:"name"`
	Deleted                    bool      `json:"deleted,omitempty"`
	Active                     bool      `json:"active,omitempty"`
	SendToNewPf                bool      `json:"sendToNewPF,omitempty"`
	Visible                    bool      `json:"visible,omitempty"`
	CurrencyISOCode            string    `json:"currencyISOCode,omitempty"`
	LanguageISOCode            string    `json:"languageISOCode,omitempty"`
	Secret                     bool      `json:"secret,omitempty"`
	AdvertiserID               uint64    `json:"advertiserId,omitempty"`
	NumberOfUnmappedCategories uint64    `json:"numberOfUnmappedCategories,omitempty"`
	NumberOfProducts           uint64    `json:"numberOfProducts"`
	Programs                   []Program `json:"programs"`
}
