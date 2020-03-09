// +build unit
// +build !integration

package wooclient

import (
	"fmt"
	"os"
	"testing"
)

type wooTestRequest struct {
	method   string
	endpoint string
	body     []byte
}

func TestWooClient(t *testing.T) {
	w, err := NewClient(
		"https://www.stillgrove.com",
		os.Getenv("WOO_KEY"),
		os.Getenv("WOO_SECRET"),
		"v3",
		true,
		4,
		2,
		2,
	)
	if err != nil {
		t.Fatalf("Get woo client %v", err)
	}

	err = testProductRequests(w)
	if err != nil {
		t.Fatalf("Test woo requests %v", err)
	}
}

func testProductRequests(w *Client) error {
	var tests = []Request{
		GetRequest{
			Endpoint: "products",
		},
		GetRequest{
			Endpoint: "products/attributes",
		},
		// fails if Perfect WooCommerce Plugin is not installed
		GetRequest{
			Endpoint: "brands",
		},
	}
	for _, t := range tests {
		_, err := t.Send(w)
		if err != nil {
			return fmt.Errorf("%s - %v", t, err)
		}
	}
	return nil
}

func testDeleteBatchRequestQueue(w *Client, productid uint64) error {
	w.PushToQueue(
		"test",
		BatchPostRequest{
			Endpoint: "products/batch",
			Delete:   []int{int(productid)},
		},
	)
	_, err := w.ExecuteRequestQueue("test", true, true)
	if err != nil {
		return fmt.Errorf("Execute delete request via queue - %v", err)
	}

	return nil
}
