// +build !unit

package awin

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

func getMapping() *feed.Mapping {
	var terms = [...]string{
		"multi",
		"5xl",
		"men",
		"women",
		"unisex",
		"white",
		"orange",
		"black",
		"grey",
		"blue",
		"red",
	}
	return &feed.Mapping{
		ColorMap: map[string][]*string{
			"multi": []*string{
				&terms[0],
			},
			"white": []*string{
				&terms[5],
			},
			"orange": []*string{
				&terms[6],
			},
			"black": []*string{
				&terms[7],
			},
			"grey": []*string{
				&terms[8],
			},
			"blue": []*string{
				&terms[9],
			},
			"red": []*string{
				&terms[10],
			},
		},
		SizeMap: map[string][]*string{
			"xxxxl": []*string{
				&terms[1],
			},
		},
		GenderMap: map[string][]*string{
			"men": []*string{
				&terms[2],
			},
			"women": []*string{
				&terms[3],
			},
			"unisex": []*string{
				&terms[4],
			},
		},
		PatternMap: map[string][]*string{},
		CatNameMap: map[string][]*string{
			"unisex": []*string{&terms[4]},
			"male":   []*string{&terms[2]},
			"female": []*string{&terms[3]},
			"men":    []*string{&terms[2]},
			"women":  []*string{&terms[3]},
		},
		ConversionMap: map[int32]*feed.Product{},
	}
}

func TestAwinFeed(t *testing.T) {
	loc, err := feed.NewLocale(
		"SE",
		"sv",
		"sv_se",
	)
	if err != nil {
		t.Fatal(err)
	}

	m := getMapping()

	fd, err := NewAwin(
		loc,
		os.Getenv("AWIN_TOKEN"),
		os.Getenv("AWIN_FEED_TOKEN"),
		m,
	)
	if err != nil {
		t.Fatal(err)
	}

	var (
		validProducts uint32
		products      []feed.Product
		foundColor    bool
	)
	for i := 0; i < 2; i++ {
		products, err = fd.Get(false)
		if err != nil {
			t.Fatal(err)
		}

		validProducts = 0
		for i := range products {
			foundColor = false

			err = products[i].Validate()
			if err != nil {
				log.Warnln(err)
				continue
			}

			for k := range m.ColorMap {
				for j := range products[i].ColorGroups {
					if products[i].ColorGroups[j] == k {
						foundColor = true
						break
					}
				}
				if foundColor {
					break
				}
			}

			if !foundColor {
				t.Fatalf("Inconsistent color mapping - %v", products[i].ColorGroups)
			}

			validProducts++
		}
	}

	log.WithFields(log.Fields{
		"Received Products": len(products),
		"Valid Products":    validProducts,
	}).Println("Test completed")
}
