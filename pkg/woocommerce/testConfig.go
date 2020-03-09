package woocommerce

import (
	"fmt"

	"stillgrove.com/gofeedyourself/pkg/feedservice/config"
	cfg "stillgrove.com/gofeedyourself/pkg/feedservice/config"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

func loadCFG() (ws cfg.TdWebsite, ColorMap, SizeMap, PatternMap, GenderMap, CatNameMap map[string][]*string, categoryMap map[string]map[string][]*int32, err error) {

	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := config.New(configPath)

	if err != nil {
		return ws, ColorMap, SizeMap, PatternMap, GenderMap, CatNameMap, categoryMap, err
	}

	_, ws, err = cfg.GetTD()
	if err != nil {
		return ws, ColorMap, SizeMap, PatternMap, GenderMap, CatNameMap, categoryMap, err
	}

	var mapNames = [...]string{
		"colors",
		"sizes",
		"patterns",
		"genders",
	}
	maps := make([]map[string][]*string, len(mapNames))

	for i := range mapNames {
		maps[i], err = cfg.GetMapping(mapNames[i])
		if err != nil {
			return ws, ColorMap, SizeMap, PatternMap, GenderMap, CatNameMap, categoryMap, fmt.Errorf("Fetching %s - %v", mapNames[i], err)
		}
	}

	categoryMap, CatNameMap, err = cfg.GetCategoryMaps()
	if err != nil {
		return ws, ColorMap, SizeMap, PatternMap, GenderMap, CatNameMap, categoryMap, err
	}

	return ws, maps[0], maps[1], maps[2], maps[3], CatNameMap, categoryMap, nil
}
