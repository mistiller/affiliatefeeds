package wpclient

import (
	"testing"
)

func TestWP(t *testing.T) {
	if true {
		t.Skip()
	}
	wp, err := New(
		"https://www.stillgrove.com",
		"v2",
		"mstiller",
		"ueldwvxxNII1Dm7E",
	)
	if err != nil {
		t.Fatal(err)
	}

	err = wp.GetMedia()
	if err != nil {
		t.Fatal(err)
	}
}
