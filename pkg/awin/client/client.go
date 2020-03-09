package awinclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

const (
	// make the timeout really generous for now because the download can take multiple hours
	requestTimeout = 1 * time.Hour
)

type Client struct {
	apiToken      string
	productsToken string
	queue         *Queue
	locale        *feed.Locale
}

func New(locale *feed.Locale, apiToken, productsToken string) (c *Client, err error) {
	var (
		exists bool
	)
	for i := range Countries {
		if Countries[i] == strings.ToUpper(locale.TwoLetterCode) {
			exists = true
		}
	}
	if !exists {
		return c, fmt.Errorf("Country not mapped - %s", locale.TwoLetterCode)
	}

	return &Client{
		apiToken:      apiToken,
		productsToken: productsToken,
		queue:         NewQueue(),
		locale:        locale,
	}, nil
}

func (c *Client) AddApiRequest(method, endpoint string, params *url.Values, payload interface{}) error {
	r, err := NewApiRequest(method, endpoint, payload, params, c.apiToken)
	if err != nil {
		return fmt.Errorf("Failed to add request to queue - %v", err)
	}
	err = c.queue.Add(r)
	if err != nil {
		return fmt.Errorf("Failed to add request to queue - %v", err)
	}
	return nil
}

func (c *Client) GetProgrammes() (programmes []Programme, err error) {
	programmes, err = GetProgrammes(c.locale.TwoLetterCode, c.apiToken)
	return programmes, err
}

func (c *Client) GetTransactions() (transactions []Transaction, err error) {
	end := time.Now()
	start := end.AddDate(0, 0, -7)
	transactions, err = GetTransactions(c.locale.TwoLetterCode, start, end, c.apiToken)
	if err != nil {
		return transactions, err
	}
	return transactions, nil
}

func (c *Client) GetFeeds() (list []Feed, err error) {
	inList, err := GetFeeds(c.productsToken)
	if err != nil {
		return list, err
	}
	for i := range inList {
		if inList[i].PrimaryRegion != c.locale.TwoLetterCode {
			continue
		}
		list = append(
			list,
			inList[i],
		)
	}

	return list, nil
}

func (c Client) GetProducts(maxCount int) (products []Product, err error) {
	var (
		activeProgrammes []Programme
		progIDs          map[uint64]struct{}
		exists, matched  bool
		inList, outList  []Feed
		results          [][]byte
		product          []Product

		key       uint64
		uniques   map[uint64]struct{}
		maxMemory uint64
		mem       runtime.MemStats
	)

	memLog("Starting to gather Awin Products", mem, &maxMemory)

	activeProgrammes, err = GetProgrammes(c.locale.TwoLetterCode, c.apiToken)
	if err != nil {
		return products, fmt.Errorf("Get active programmes - %v", err)
	}

	if len(activeProgrammes) == 0 {
		return products, nil
	}

	progIDs = make(map[uint64]struct{})
	for i := range activeProgrammes {
		_, exists = progIDs[activeProgrammes[i].ProgrammeInfo.ID]
		if !exists {
			progIDs[activeProgrammes[i].ProgrammeInfo.ID] = struct{}{}
		}
	}

	inList, err = c.GetFeeds()
	if err != nil {
		return products, fmt.Errorf("Get list - %v", err)
	}

	matched, err = c.enqueue(inList, progIDs, maxCount)
	if err != nil {
		return products, fmt.Errorf("Enqueue requests - %v", err)
	}
	if !matched {
		fList := ""
		for i := range inList {
			fList += inList[i].AdvertiserName
			if i < len(inList)-1 {
				fList += ", "
			}
		}
		pList := ""
		for j := range activeProgrammes {
			pList += activeProgrammes[j].ProgrammeInfo.Name
			if j < len(activeProgrammes[j].ProgrammeInfo.Name)-1 {
				pList += ", "
			}
		}
		log.WithFields(
			log.Fields{
				"Programmes": pList,
				"Feeds":      fList,
			},
		).Warnln("Couldn't find feeds for programmes")
		return products, nil
	}

	for i := range outList {
		err = c.queue.Add(
			NewProductRequest(
				outList[i].URL,
				c.productsToken,
				maxCount,
			),
		)
		if err != nil {
			return products, fmt.Errorf("Failed to add request to queue - %v", err)
		}
	}

	memLog("Downloading Awin Products", mem, &maxMemory)
	results, err = c.Execute(false)
	if err != nil {
		return products, fmt.Errorf("Query feeds - %v", err)
	}

	uniques = make(map[uint64]struct{})
	for i := range results {
		memLog(fmt.Sprintf("Processing Feed %d", i), mem, &maxMemory)
		if results[i] == nil {
			continue
		}
		err = json.Unmarshal(results[i], &product)
		if err != nil {
			log.Warnf("Unmarshal Products - %v", err)
			continue
		}
		for j := range product {
			key, err = product[j].Key()
			if err != nil {
				continue
			}
			_, exists = uniques[key]
			if exists {
				continue
			}
			err = product[j].AttachProgrammes(activeProgrammes)
			if err != nil {
				log.WithFields(
					log.Fields{
						"Product Name": product[j].ProductName,
						"Error":        err,
					},
				).Debugln("Failed to attach programme")
				continue
			}

			err = product[j].Validate()
			if err != nil {
				log.WithFields(
					log.Fields{
						"Product Name": product[j].ProductName,
						"Error":        err,
					},
				).Debugln("Validation Error")
				continue
			}
			products = append(products, product[j])
			uniques[key] = struct{}{}

			if maxCount > 0 && len(products) >= maxCount {
				return products, nil
			}
		}
	}
	memLog("All feeds processed", mem, &maxMemory)

	return products, nil
}

func (c *Client) enqueue(in []Feed, activeProgrammes map[uint64]struct{}, maxRows int) (matched bool, err error) {
	for i := range in {
		if in[i].PrimaryRegion != c.locale.TwoLetterCode {
			continue
		}
		for key := range activeProgrammes {
			if in[i].AdvertiserID == key {
				r := NewProductRequest(
					in[i].URL,
					c.productsToken,
					maxRows,
				)
				err = c.queue.Add(r)
				if err != nil {
					return matched, err
				}

				matched = true
			}
		}
	}

	return matched, nil
}

func (c *Client) Execute(strict bool) ([][]byte, error) {
	result, err := c.queue.Execute(strict)
	if err != nil {
		return result, fmt.Errorf("Awin: Execute Request Queue - %v", err)
	}

	return result, nil
}
