package testsuite

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"stillgrove.com/gofeedyourself/pkg/collection"
	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"stillgrove.com/gofeedyourself/pkg/feedservice/config"
	f "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	td "stillgrove.com/gofeedyourself/pkg/tradedoubler"
	gwc "stillgrove.com/gofeedyourself/pkg/woocommerce"
)

type FeedTestSuite struct {
	suite.Suite
	token       string
	ColorMap    map[string][]*string
	SizeMap     map[string][]*string
	PatternMap  map[string][]*string
	GenderMap   map[string][]*string
	CatNameMap  map[string][]*string
	categoryMap map[string]map[string][]*int32
	gwc         gwc.WooConnection
	testFeed    *td.TestFeed
	testPM      *f.ProductMap
}

type requestQueue struct {
	Lang   string                   `json:"lang"`
	Create []map[string]interface{} `json:"create,omitempty"`
	Update []map[string]interface{} `json:"update,omitempty"`
	Delete []int                    `json:"delete,omitempty"`
}

// SetupTest makes sure all the credentials are properly loaded and the feedservice can be tested
func (s *FeedTestSuite) SetupTest() {
	var (
		err error
	)

	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := config.New(configPath)
	assert.Nil(s.T(), err)

	_, ws, err := cfg.GetTD()
	assert.Nil(s.T(), err)

	s.token = ws.Token

	var mapNames = [...]string{
		"colors",
		"sizes",
		"patterns",
		"genders",
	}
	maps := make([]map[string][]*string, len(mapNames))

	for i := range mapNames {
		maps[i], err = cfg.GetMapping(mapNames[i])
		assert.Nil(s.T(), err)
	}

	s.ColorMap = maps[0]
	s.SizeMap = maps[1]
	s.PatternMap = maps[2]
	s.GenderMap = maps[3]

	s.categoryMap, s.CatNameMap, err = cfg.GetCategoryMaps()
	assert.Nil(s.T(), err)

	s.gwc, err = gwc.NewWooConnection(
		"https://www.stillgrove.com",
		os.Getenv("WOO_KEY"),
		os.Getenv("WOO_SECRET"),
		"sv_se",
	)
	assert.Nil(s.T(), err)

	s.testFeed, err = td.NewTestFeed(
		s.token,
		"sv",
		s.ColorMap,
		s.PatternMap,
		s.SizeMap,
		s.GenderMap,
		s.CatNameMap,
	)
	assert.Nil(s.T(), err)

	q := f.NewQueueFromFeeds([]f.Feed{s.testFeed}, false)
	s.testPM, err = q.GetPM(true)
	assert.Nil(s.T(), err)
}

// TestSetup checks whether the setup has been completed successfully
func (s *FeedTestSuite) TestSetup() {
	assert.NotEqual(s.T(), s.token, "")

	l := len(s.ColorMap) * len(s.ColorMap) * len(s.SizeMap) * len(s.ColorMap) * len(s.PatternMap) * len(s.GenderMap) * len(s.CatNameMap) * len(s.categoryMap)
	assert.NotEqual(s.T(), l, 0)

	assert.NotNil(s.T(), s.testFeed, s.gwc)
}

// TestTradedoubler checks whether the a production-ready Tradedoubler feed can be downloaded
func (s *FeedTestSuite) TestTradedoubler() {
	s.T().Skip()

	loc, err := feed.NewLocale("SE", "sv", "sv_se")
	assert.Nil(s.T(), err)

	td, err := td.NewFeed(
		loc,
		s.token,
		"",
		"",
		"",
		s.ColorMap,
		s.PatternMap,
		s.SizeMap,
		s.GenderMap,
		s.CatNameMap,
		"sv",
	)
	assert.Nil(s.T(), err)

	products, err := td.Get(false)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), len(products), 0, "No products downloaded")

	for i := range products {
		err = products[i].Validate()
		assert.Nil(s.T(), err)
	}
}

func (s *FeedTestSuite) TestFeed() {
	var err error

	products, np, nf, nc := s.testPM.Get()
	log.Printf("%d Products from %d feeds w/ %d categories\n", np, nf, nc)
	for k := range products {
		products[k].Update()
		err = products[k].Validate()
		assert.Nil(s.T(), err)
	}

	log.Infoln("Prepare Update")
	err = s.gwc.PrepareUpdate(s.testPM, s.categoryMap, false, false)
	assert.Nil(s.T(), err)
}

func (s *FeedTestSuite) TestUpdate() {
	mappings, err := s.gwc.PrepareMappings(s.testPM, s.categoryMap, false)
	assert.Nil(s.T(), err)

	testPM, err := gwc.PMFromPM(s.testPM, &mappings)
	assert.Nil(s.T(), err)

	p, _, _ := testPM.Get()
	var str string
	for i := range p {
		str = ""
		for j := range p[i].Attributes {
			if p[i].Attributes[j].Name == "Color Group" || p[i].Attributes[j].Name == "Color" {
				str += fmt.Sprintf("%v", p[i].Attributes[j].Options)
			}
		}
		log.Infoln(str)
	}

	oldProductMap, err := s.gwc.GetOldProductMap(false)
	assert.Nil(s.T(), err)

	create, update, delete, err := testPM.GetGroups(oldProductMap)
	assert.Nil(s.T(), err)

	testPM.Flush()

	assert.NotEqual(s.T(), len(create)+len(update)+len(delete), 0)

	err = s.gwc.BuildDeleteProductQueue(delete)
	assert.Nil(s.T(), err)

	err = s.gwc.BuildCreateUpdateProductQueue(create, update, false)
	assert.Nil(s.T(), err)

	req := new(requestQueue)
	requests, err := s.gwc.Connection.ViewRequestQueue()
	assert.Nil(s.T(), err)

	nChange := 0
	for i := range requests {
		err = json.Unmarshal(requests[i], req)
		assert.Nil(s.T(), err)

		nChange += len(req.Create) + len(req.Update)
	}
	assert.NotEqual(s.T(), nChange, 0, "No products to create or update")
}

func (s *FeedTestSuite) TestMappings() {
	var testCases = []string{
		"REDDISH",
		"firecracker",
		"BLEEN",
		"oLIVe",
	}
	mapped, err := collection.MapAttributes(testCases, s.ColorMap, "", true)
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), mapped, []string{"red"})
}

func (s *FeedTestSuite) TestCategories() {
	var (
		err   error
		nCats int
	)

	q := f.NewQueueFromFeeds([]f.Feed{s.testFeed}, false)
	pm, err := q.GetPM(true)
	assert.Nil(s.T(), err)

	products, _, _, _ := pm.Get()

	temp := new(gwc.FeedProduct)
	for k := range products {
		temp = &gwc.FeedProduct{
			*products[k],
		}

		cats, err := gwc.GetWCCategories(temp.ProviderCategories, s.categoryMap, true)
		for i := range cats {
			assert.NotEqual(s.T(), cats[i], 0)
		}
		if err != nil {
			log.Println(err)
			continue
		}
		if len(cats) == 0 {
			continue
		}
		nCats += len(cats)

		wp := new(gwc.Product)
		err = wp.AddCategories(cats)
		assert.Nil(s.T(), err)
		assert.NotEqual(s.T(), len(wp.Categories), 0)
	}

	assert.NotEqual(s.T(), nCats, 0)
}
