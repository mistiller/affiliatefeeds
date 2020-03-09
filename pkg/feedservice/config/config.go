package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"stillgrove.com/gofeedyourself/pkg/collection"

	"gopkg.in/yaml.v2"
	"stillgrove.com/gofeedyourself/pkg/googlesheets"
)

var (
	EnvVars = []string{
		"WOO_KEY",
		"WOO_SECRET",
		"DYNAMO_ID",
		"DYNAMO_SECRET",
		"FTP_HOST",
		"FTP_USER",
		"FTP_PASS",
		"FTP_PORT",
		"EMAIL_PW",
		"AWIN_TOKEN",
		"AWIN_FEED_TOKEN",
	}
)

type tdFeed struct {
	Name string `yaml:"name"`
	ID   int    `yaml:"id"`
}

// TdWebsite contains all the configs from the Tradedoubler Site
// such as: tokens, feed ids
type TdWebsite struct {
	Name  string `yaml:"name"`
	Token string
	Feeds map[int]tdFeed
}

type tdConfig struct {
	ConversionTable string    `yaml:"conversionTable"`
	Website         TdWebsite `yaml:"website"`
}
type wooConfig struct {
	Domain string `yaml:"domain"`
	key    string
	secret string
}
type dynamoConfig struct {
	ID           string
	secret       string
	token        string
	ProductTable string `yaml:"productTable"`
}
type gsheetConfig struct {
	ID        string `yaml:"id"`
	CellRange string `yaml:"range"`
}
type ftpConfig struct {
	host     string
	username string
	password string
	port     int
}
type emailConfig struct {
	Name     string `yaml:"name"`
	Server   string `yaml:"server"`
	password string
}
type awinConfig struct {
	apiToken  string
	feedToken string
}

// File contains all settings for a FeedService instance
type File struct {
	Country   string                  `yaml:"country"`
	Locale    string                  `yaml:"locale"`
	Language  string                  `yaml:"language"`
	Time      string                  `yaml:"time"`
	CleanDays []string                `yaml:"clean_days"`
	Woo       wooConfig               `yaml:"woocommerce"`
	TD        tdConfig                `yaml:"tradedoubler"`
	Dynamo    dynamoConfig            `yaml:"dynamodb"`
	GSheet    map[string]gsheetConfig `yaml:"gsheet"`
	email     emailConfig             `yaml:"email"`
	ftp       ftpConfig
	Awin      awinConfig
}

// New returns a pointer to a config object
func New(filePath string) (cfg *File, err error) {
	cfg = new(File)

	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return cfg, err
	}

	names := append(EnvVars, "TD_TOKEN_"+cfg.TD.Website.Name)
	envs, err := getEnvs(names)
	if err != nil {
		return cfg, err
	}

	cfg.TD.Website.Token = envs["TD_TOKEN_"+cfg.TD.Website.Name]
	cfg.Awin.apiToken = envs["AWIN_TOKEN"]
	cfg.Awin.feedToken = envs["AWIN_FEED_TOKEN"]

	cfg.Woo.key = envs["WOO_KEY"]
	cfg.Woo.secret = envs["WOO_SECRET"]

	cfg.Dynamo.ID = envs["DYNAMO_ID"]
	cfg.Dynamo.secret = envs["DYNAMO_SECRET"]

	ftport, _ := strconv.Atoi(envs["FTP_PORT"])
	cfg.ftp = ftpConfig{
		host:     envs["FTP_HOST"],
		username: envs["FTP_USER"],
		password: envs["FTP_PASS"],
		port:     ftport,
	}

	cfg.email.password = envs["EMAIL_PW"]

	return cfg, nil
}

// NewVSF returns a pointer to a config object with fewer environment variables
func NewVSF(filePath string) (cfg *File, err error) {
	cfg = new(File)

	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return cfg, err
	}

	envs, err := getEnvs(
		[]string{
			"TD_TOKEN_" + cfg.TD.Website.Name,
			"DYNAMO_ID",
			"DYNAMO_SECRET",
			"AWIN_TOKEN",
			"AWIN_FEED_TOKEN",
		},
	)
	if err != nil {
		return cfg, err
	}

	cfg.TD.Website.Token = envs["TD_TOKEN_"+cfg.TD.Website.Name]

	cfg.Awin.apiToken = envs["AWIN_TOKEN"]
	cfg.Awin.feedToken = envs["AWIN_FEED_TOKEN"]

	cfg.Dynamo.ID = envs["DYNAMO_ID"]
	cfg.Dynamo.secret = envs["DYNAMO_SECRET"]

	return cfg, nil
}

// SetHost let's you override the host from the config file
func (cfg *File) SetHost(newHost string) {
	cfg.Woo.Domain = newHost
}

// GetEmail returns password for the notification email address
func (cfg *File) GetEmail() (name string, server string, pass string) {
	return cfg.email.Name, cfg.email.Server, cfg.email.password
}

// GetLocale returns the locale set in the config file -error if not set
func (cfg *File) GetLocale() (country, locale, language string, err error) {
	if cfg.Locale == "" {
		return cfg.Country, cfg.Locale, cfg.Language, fmt.Errorf("Locale not set")
	}
	if cfg.Language == "" {
		return cfg.Country, cfg.Locale, cfg.Language, fmt.Errorf("Language not set")
	}
	return cfg.Country, cfg.Locale, cfg.Language, nil
}

// GetCategoryMaps return the mapping table between different category names and WC ids,
// plus a list oft the names themselves
func (cfg *File) GetCategoryMaps() (catMap map[string]map[string][]*int32, CatNameMap map[string][]*string, err error) {
	catMap = map[string]map[string][]*int32{
		"m": make(map[string][]*int32),
		"w": make(map[string][]*int32),
		"u": make(map[string][]*int32),
	}

	sheet, datarange, err := cfg.GetGSheet("categories")

	data, err := googlesheets.LoadFromGSheet(sheet, datarange)
	if err != nil {
		return catMap, CatNameMap, err
	}

	var (
		name, gender string
		id64         int64
	)
	for _, row := range data {
		if len(row) < 3 {
			continue
		}
		name = strings.ToLower(row[0].(string))
		id64, err = strconv.ParseInt(row[1].(string), 10, 32)
		id := int32(id64)

		if err != nil {
			return catMap, CatNameMap, fmt.Errorf("Unable to parse config file - %v", row)
		}
		if row[2] != nil {
			gender = row[2].(string)
		} else {
			gender = "u"
		}

		catMap[gender][name] = append(catMap[gender][name], &id)
	}

	var exist bool
	CatNameMap = make(map[string][]*string)
	for gender := range catMap {
		for key := range catMap[gender] {
			_, exist = CatNameMap[key]
			if !exist {
				k := key
				CatNameMap[k] = []*string{&k}
			}
		}
	}

	return catMap, CatNameMap, nil
}

//GetMapping return the mapping table for color simplifications
func (cfg *File) GetMapping(name string) (mapping map[string][]*string, err error) {
	sheet, datarange, err := cfg.GetGSheet(name)
	if err != nil {
		return mapping, err
	}

	data, err := googlesheets.LoadFromGSheet(sheet, datarange)
	if err != nil {
		return mapping, err
	}

	mapping = make(map[string][]*string)
	var k string
	var exist bool
	for _, row := range data {
		if len(row) < 2 {
			continue
		}
		s := strings.ToLower(row[1].(string))
		k = strings.ToLower(row[0].(string))
		_, exist = mapping[k]
		if !exist {
			mapping[k] = []*string{
				&s,
			}
			continue
		}
		mapping[k] = append(mapping[k], &s)
	}

	return mapping, nil
}

// GetTD returns ConversionTable, Website domain, and error
// for a Tradedoubler page
func (cfg *File) GetTD() (conversionTable string, website TdWebsite, err error) {
	if cfg.TD.ConversionTable == "" {
		return conversionTable, website, fmt.Errorf("Couldn't load TD config")
	}
	return cfg.TD.ConversionTable, cfg.TD.Website, nil
}

// GetAwin retuns Awin credentials
func (cfg *File) GetAwin() (apiToken, feedToken string, err error) {
	if cfg.Awin.apiToken == "" || cfg.Awin.feedToken == "" {
		return apiToken, feedToken, fmt.Errorf("Couldn't find Awin token")
	}
	return cfg.Awin.apiToken, cfg.Awin.feedToken, nil
}

// GetWoo returns domain, key, secret, and error for a WooCommerce page
func (cfg *File) GetWoo() (string, string, string, error) {
	return cfg.Woo.Domain, cfg.Woo.key, cfg.Woo.secret, nil
}

// GetDynamo returns ID, Secret, Token, ProductTable, and error
func (cfg *File) GetDynamo() (id, secret, productTable string, err error) {
	if collection.AnyEmpty(
		[]*string{
			&cfg.Dynamo.ID,
			&cfg.Dynamo.secret,
			&cfg.Dynamo.ProductTable,
		},
	) {
		return id, secret, productTable, fmt.Errorf("Empty fields in dynamo config")
	}
	return cfg.Dynamo.ID, cfg.Dynamo.secret, cfg.Dynamo.ProductTable, nil
}

// GetGSheet returns tableID, datarange, and error
func (cfg *File) GetGSheet(name string) (id string, cells string, err error) {
	if cfg.GSheet[name].ID == "" || cfg.GSheet[name].CellRange == "" {
		return id, cells, fmt.Errorf("CFG - Name not found in config")
	}

	return cfg.GSheet[name].ID, cfg.GSheet[name].CellRange, nil
}

// GetFTP returns host, port, username, password, and error
func (cfg *File) GetFTP() (string, int, string, string, error) {
	if cfg.ftp.host == "" {
		return cfg.ftp.host, cfg.ftp.port, cfg.ftp.username, cfg.ftp.password, errors.New("Couldn't load FTP config")
	}
	return cfg.ftp.host, cfg.ftp.port, cfg.ftp.username, cfg.ftp.password, nil
}

func getEnvs(names []string) (map[string]string, error) {
	variables := make(map[string]string, len(names))
	for _, n := range names {
		variables[n] = os.Getenv(n)
		if variables[n] == "" {
			return variables, fmt.Errorf("Couldn't find env variable: %s", n)
		}
	}
	return variables, nil
}
