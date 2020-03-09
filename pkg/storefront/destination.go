package storefront

import (
	"fmt"
	"strconv"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

type Destination struct {
	Name         string
	Link         string
	Sizes        []string
	SalesPrice   float64
	RegularPrice float64
}

func GetDestination(p *feed.Product) (dest *Destination, err error) {
	var (
		price, lowest, highest float64
		activeID               int
		matched                bool
	)
	for i := range p.Retailers {
		price, err = strconv.ParseFloat(p.Retailers[i].Price, 32)
		if err != nil {
			continue
		}
		if lowest == 0.0 {
			lowest = price
		}

		if price <= lowest && !p.Retailers[i].IsCrawler {
			activeID = i
			lowest = price
			matched = true
		}
		if price > highest {
			highest = price
		}
	}

	if !matched {
		return dest, fmt.Errorf("No destination found")
	}

	dest = &Destination{
		Name:         p.Retailers[activeID].Name,
		Link:         p.Retailers[activeID].Link,
		Sizes:        p.Retailers[activeID].Sizes,
		SalesPrice:   lowest,
		RegularPrice: highest,
	}

	return dest, nil
}
