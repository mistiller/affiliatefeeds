package wooclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/publicsuffix"
)

const (
	Version        = "1.0.0"
	UserAgent      = "Feedservice API Client-Golang/" + Version
	HashAlgorithm  = "HMAC-SHA256"
	Prefix         = "/wp-json/wc/"
	ErrorLimit     = 10
	RequestRetries = 2
	requestTimeout = 10 * time.Minute
)

var (
	LoadPageSize = 100
)

// Client interfaces with the WooCommerce backend
type Client struct {
	initialized bool
	//client                *wc.Client
	maxRetries            int
	batchStrideSize       int // defines the size of one chunk for the batch upload
	maxConcurrentRequests int // defines how many requests can be sent concurrently
	requestQueue          map[string][]Request
	domain, key, secret   string
	rawClient             *http.Client
	Timeout               time.Duration
	VerifySSL             bool
	QueryStringAuth       string
	OauthTimestamp        time.Time
	Version               string
	storeURL              *url.URL
}

// NewClient takes in the credentials and returns an initialized client
func NewClient(domain, key, secret, version string, verifySSL bool, productsPerBatch, maxConcurrentRequests, maxRetries int) (w *Client, err error) {
	rawClient := new(http.Client)
	rawClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: verifySSL},
	}
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	rawClient.Jar = jar

	w = &Client{
		initialized:           true,
		maxRetries:            maxRetries,
		batchStrideSize:       productsPerBatch,
		maxConcurrentRequests: maxConcurrentRequests,
		domain:                domain,
		key:                   key,
		secret:                secret,
		rawClient:             rawClient,
	}

	storeURL, err := url.Parse(domain)
	if err != nil {
		return w, err
	}

	switch version {
	case "v1":
		w.Version = "v1"
	case "v2":
		w.Version = "v1"
	case "v3":
		w.Version = "v3"
	default:
		return w, fmt.Errorf("Please select either v1, v2, or v3 for the API version")
	}
	storeURL.Path = Prefix + w.Version + "/"
	w.storeURL = storeURL

	if w.OauthTimestamp.IsZero() {
		w.OauthTimestamp = time.Now()
	}

	w.requestQueue = map[string][]Request{
		"createupdate": []Request{},
		"delete":       []Request{},
		"brandmap":     []Request{},
		"oldProducts":  []Request{},
	}

	return w, nil
}

// GetAllProducts returns all products from the WC backend
func (w *Client) GetAllProducts(locale string, verbose bool) (currentProducts []Product, err error) {
	if w.initialized == false {
		return currentProducts, fmt.Errorf("Please initialize with your credentials first. WooConnection.Init()")
	}

	const queueName = "read"

	endpoint := "products"

	totalNumProducts, err := w.GetNumItems(endpoint, locale) //get total number of items from product endpoint
	if err != nil {
		return currentProducts, fmt.Errorf("Get number of items - %v", err)
	}

	if LoadPageSize > totalNumProducts {
		LoadPageSize = totalNumProducts
	}

	for offset := 0; offset < totalNumProducts; offset += LoadPageSize {
		w.PushToQueue(
			queueName,
			GetRequest{
				Endpoint: endpoint,
				Params: url.Values{
					"offset": []string{
						strconv.Itoa(offset),
					},
					"per_page": []string{
						strconv.Itoa(LoadPageSize),
					},
				},
			},
		)
	}
	rawResponse, err := w.ExecuteRequestQueue(queueName, true, verbose)
	if err != nil {
		return currentProducts, err
	}

	//currentProducts = make([]Product, 0)
	for i := range rawResponse {
		if len(rawResponse[i]) < 1 {
			continue
		}
		var p []Product
		err = json.Unmarshal(rawResponse[i], &p)
		if err != nil {
			continue
		}
		for j := range p {
			currentProducts = append(currentProducts, p[j])
		}
	}

	w.requestQueue[queueName] = nil

	return currentProducts, nil
}

// PurgeProducts deletes all the products from the woo commerce backend
// Remember: Does not remove the image assets from the server!
func (w *Client) PurgeProducts(locale string, verbose bool) error {
	if w.initialized == false {
		return fmt.Errorf("Please initialize with your credentials first. WooConnection.Init()")
	}

	products, err := w.GetAllProducts(locale, false)
	if err != nil {
		return err
	}

	endpoint := "products/batch"

	var r = BatchPostRequest{
		Endpoint: endpoint,
	}

	// Preparing queue of product ids to be deleted
	for i := range products {
		r.Delete = append(r.Delete, int(products[i].ID))

		if i%w.maxConcurrentRequests == 0 || i >= len(products)-1 {
			w.PushToQueue("delete", r)
			r = BatchPostRequest{
				Endpoint: endpoint,
			}
		}
	}

	_, err = w.ExecuteRequestQueue("delete", false, verbose)
	if err != nil {
		return err
	}

	return nil
}

// PushToQueue appends a WooREquest to the queue to later be executed
func (w *Client) PushToQueue(name string, r Request) {
	if w.requestQueue == nil {
		w.requestQueue = make(map[string][]Request)
	}
	_, exist := w.requestQueue[name]
	if !exist {
		w.requestQueue[name] = make([]Request, 0)
	}
	w.requestQueue[name] = append(w.requestQueue[name], r)
}

// ExecuteRequestQueue executes all the request that were pushed before and returns an array of the raw responses as bytes
// if strict: returns on any error; else: finishes regardless of errors
func (w *Client) ExecuteRequestQueue(name string, strict, verbose bool) (rawResponse [][]byte, err error) {
	if len(w.requestQueue) == 0 {
		return rawResponse, fmt.Errorf("Request Queue empty")
	}
	_, exist := w.requestQueue[name]
	if !exist {
		return rawResponse, fmt.Errorf("No request queue with name %s", name)
	}

	if len(w.requestQueue[name]) == 0 {
		return rawResponse, nil
	}
	var wg sync.WaitGroup

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	input := make(chan Request, len(w.requestQueue[name]))
	output := make(chan []byte, len(w.requestQueue[name]))

	var errs uint64
	// Increment waitgroup counter and create go routines
	for i := 0; i < w.maxConcurrentRequests; i++ {
		wg.Add(1)
		go func(input chan Request, output chan []byte) {
			defer wg.Done()

			for req := range input {
				var resp []byte
				for i := 1; i <= RequestRetries; i++ {
					resp, err = req.Send(w)
					if err == nil {
						break
					}
					if strict {
						cancel()
						log.WithFields(
							log.Fields{
								"Queue": name,
								"Error": err,
							},
						).Fatalln("Request error")
					}
					log.WithFields(
						log.Fields{
							"Queue": name,
							"Error": err,
						},
					).Debugln("Request error")
					if i == RequestRetries {
						atomic.AddUint64(&errs, 1)
					}
				}
				select {
				case <-ctx.Done():
					output <- nil
				default:
					output <- resp
				}
			}
		}(input, output)
	}

	// Producer: load up input channel with jobs
	for _, job := range w.requestQueue[name] {
		input <- job
	}

	log.WithField("Requests", len(w.requestQueue[name])).Info("Queue was scheduled")

	close(input)

	rawResponse = make([][]byte, len(w.requestQueue[name]))
	var i int
	for res := range output {
		rawResponse[i] = res
		i++
		if verbose == true && (i%10 == 0 || i == len(w.requestQueue[name])) {
			progressBar(i, len(w.requestQueue[name]))
			time.Sleep(50 * time.Millisecond)
		}

		/*if errs > ErrorLimit {
			return rawResponse, fmt.Errorf("More than %d errors in request queue", ErrorLimit)
		}*/

		select {
		case <-ctx.Done():
			close(output)
			cancel()
			log.Debugln("Go routine canceled")
			break
		default:
			if i >= len(w.requestQueue[name]) {
				close(output)
				cancel()
				break
			}
		}
	}

	wg.Wait()

	w.requestQueue[name] = nil

	return rawResponse, nil
}

// ViewRequestQueue returns the marshalled requests as they will be sent by ExecuteRequestQueue
func (w *Client) ViewRequestQueue() (output [][]byte, err error) {
	var (
		temp       []byte
		errorCount int
	)

	for name := range w.requestQueue {
		for i := range w.requestQueue[name] {
			temp, err = json.Marshal(w.requestQueue[name][i])
			if err != nil {
				fmt.Println(err)
				errorCount++
				if errorCount > ErrorLimit {
					return output, err
				}
			}
			output = append(output, temp)
		}
	}

	return output, nil
}

// GetNumItems returns the total number of items (products, categories) from the repsonse header of a given endpoint
func (w *Client) GetNumItems(endpoint, locale string) (int, error) {
	if w.initialized == false {
		return 0, errors.New("Please initialize with your credentials first. WooConnection.Init()")
	}

	reqURL := w.domain + "/wp-json/wc/v3/" + endpoint

	if strings.HasPrefix(w.domain, "https") == true {
		params := make(url.Values)
		params.Add("consumer_key", w.key)
		params.Add("consumer_secret", w.secret)
		params.Add("per_page", "1")
		params.Add("lang", locale)

		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(w.key, w.secret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	rsp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	r, _ := httputil.DumpRequestOut(req, false)
	if rsp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Failed: %s\n %s", string(r), rsp.Status)
	}

	numP := rsp.Header.Get("X-WP-Total")
	np, err := strconv.ParseInt(numP, 0, 32)
	if err != nil {
		return 0, fmt.Errorf("Unable to gather total number of products - %v", err)
	}

	return int(np), nil
}
