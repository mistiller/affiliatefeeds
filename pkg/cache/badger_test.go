// +build unit
// +build !integration

package cache

import (
	"testing"
	"time"

	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

func TestBadgerCache(t *testing.T) {
	path := helpers.FindFolderDir("gofeedyourself") + "/cache/test/"

	cache, err := NewBadgerCache(path, 5*time.Minute)
	if err != nil {
		t.Errorf("%v", err)
	}
	//defer cache.Close()

	var payload = map[string][]byte{
		"test": []byte("abc"),
	}
	err = cache.Store(payload)
	if err != nil {
		t.Errorf("%v", err)
	}
	val, err := cache.Load("test")
	if err != nil {
		t.Errorf("%v", err)
	}

	if string(val) != "abc" {
		t.Errorf("Failed to retrieve the expected value from the cache")
	}
}
