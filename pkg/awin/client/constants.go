package awinclient

const (
	ConcurrentRequests = 4
	RequestRetries     = 2
	FeedList           = "https://productdata.awin.com/datafeed/list/apikey"
	BaseURL            = "https://api.awin.com"
)

var (
	Countries = []string{
		"GB",
		"SE",
	}
	Languages = []string{
		"en",
		"sv",
		"sv_se",
	}
)
