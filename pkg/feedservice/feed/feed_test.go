// +build unit
// +build !integration

package feed

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestCollate(t *testing.T) {
	f := NewTestFeed("QueueTest")
	products, _ := f.Get(false)

	part1 := products[0]
	part2 := products[1]

	err := part1.MergeWith(&part2)
	if err != nil {
		t.Fatalf("Failed to merge: %v", err)
	}
	if len(part1.Retailers) == len(products[0].Retailers) {
		t.Fatalf("Failed to add retailers: %v", err)
	}
}

func TestQueue(t *testing.T) {
	q := NewQueueFromFeeds(
		[]Feed{
			NewTestFeed("QueueTest"),
		},
		false,
	)
	pm, err := q.GetPM(true)
	if err != nil {
		t.Fatalf("Product scrambled: %v", err)
	}

	products, np, nf, nc := pm.Get()
	log.Infof("%d Products from %d feeds w/ %d categories\n", np, nf, nc)
	for k := range products {
		products[k].Update()
		err = products[k].Validate()
		if err != nil {
			t.Fatalf("Product scrambled: %s - %v", products[k].Name, err)
		}
	}
}
