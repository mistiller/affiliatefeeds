// +build unit
// +build !integration

package config

import (
	"testing"

	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

func TestConfig(t *testing.T) {
	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, _, _, err = cfg.GetLocale()
	if err != nil {
		t.Fatalf("Failed to load locale config")
	}

	_, _, err = cfg.GetTD()
	if err != nil {
		t.Fatalf("Failed to load TD config")
	}

	_, _, _, err = cfg.GetWoo()
	if err != nil {
		t.Fatalf("Failed to load Woo config")
	}

	// Skip the other tests in case we are in a hurry or want to omitt Google Sheets
	if testing.Short() {
		t.Skip("skipping gsheet testing in short mode")
	}

	_, _, err = cfg.GetCategoryMaps()
	if err != nil {
		t.Fatalf("Failed to load category config from gsheets")
	}

	var mappings = []string{
		"colors",
		"sizes",
		"patterns",
		"genders",
	}
	m := make(map[string][]*string)
	for _, name := range mappings {
		m, err = cfg.GetMapping(name)
		if err != nil || len(m) == 0 {
			t.Fatalf("Failed to load %s config from gsheets", name)
		}

		uniques := make(map[string]struct{})
		exist := false
		for _, values := range m {
			for i := range values {
				val := *values[i]
				_, exist = uniques[val]
				if !exist {
					uniques[val] = struct{}{}
				}
			}
		}
		if len(uniques) < 2 {
			t.Fatalf("Mapping produced too few terms - %v", uniques)
		}
	}
}
