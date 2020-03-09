package feed

// Mapping contains all the relevant mapping tables for td products
type Mapping struct {
	ColorMap      map[string][]*string
	SizeMap       map[string][]*string
	GenderMap     map[string][]*string
	PatternMap    map[string][]*string
	CatNameMap    map[string][]*string
	ConversionMap map[int32]*Product
}

/*
func NewMapping(colmap, sizemap, gendermap, patternmap, catnamemap map[string][]*string) (m *Mapping, err error) {
	return &Mapping{
		ColorMap:   colmap,
		SizeMap:    sizemap,
		GenderMap:  gendermap,
		PatternMap: patternmap,
		CatNameMap: catnamemap,
	}, nil
}
*/
