package awinclient

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gocarina/gocsv"
)

type Feed struct {
	AdvertiserID     uint64 `csv:"Advertiser ID"`
	AdvertiserName   string `csv:"Advertiser Name"`
	PrimaryRegion    string `csv:"Primary Region"`
	MembershipStatus string `csv:"Membership Status"`
	FeedID           uint64 `csv:"Feed ID"`
	FeedName         string `csv:"Feed Name"`
	Language         string `csv:"Language"`
	Vertical         string `csv:"Vertical"`
	LastImported     string `csv:"Last Imported"`
	LastChecked      string `csv:"Last Checked"`
	NoOfProducts     string `csv:"No of products"`
	URL              string `csv:"URL"`
}

func GetFeeds(token string) (list []Feed, err error) {
	resp, err := http.Get(FeedList + "/" + token)
	if err != nil {
		return list, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return list, err
	}

	err = gocsv.UnmarshalBytes(b, &list)
	if err != nil {
		return list, err
	}

	u := new(url.URL)
	for i := range list {
		u, err = url.Parse(list[i].URL)
		if err != nil {
			return list, err
		}
		list[i].URL = u.String()
	}
	return list, nil
}
