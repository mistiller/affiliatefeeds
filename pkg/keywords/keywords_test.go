// +build ignore

package keywords

import (
	"log"
	"os"
	"testing"

	w "stillgrove.com/gofeedyourself/pkg/woocommerce"
)

func TestKeywords(t *testing.T) {
	wc, err := w.NewWooConnection(
		"https://www.stillgrove.com",
		os.Getenv("WOO_KEY"),
		os.Getenv("WOO_SECRET"),
		"sv_se",
	)
	if err != nil {
		t.Fatal(err)
	}

	kw, err := GenerateKeywordsFromWoo(wc.Connection)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(kw)
}
