// +build !unit
// +build integration

package tradedoublerclient

import (
	"log"
	"testing"

	"stillgrove.com/gofeedyourself/pkg/feedservice/config"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

const (
	BatchSize  = 500
	SampleSize = 2000
)

func TestProductFactory(t *testing.T) {

	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := config.New(configPath)
	if err != nil {
		t.Fatal(err)
	}

	_, ws, err := cfg.GetTD()
	if err != nil {
		t.Fatal(err)
	}

	c, err := NewConnection(ws.Token)
	if err != nil {
		t.Fatalf("Failed to initialize td connection - %v", err)
	}

	nProducts, err := c.InitProductFactory(BatchSize, "sv")
	if err != nil {
		t.Fatal(err)
	}

	var (
		products []Product
		counter  uint64
		done     bool
	)
	for !done {
		products, done, err = c.ProductFactoryNext()
		if done {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		counter += uint64(len(products))
		if counter >= SampleSize {
			done = true
		}

		log.Printf("Retrieved %d / %d products\n", counter, nProducts)
	}
}
