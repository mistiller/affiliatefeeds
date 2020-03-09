package awinclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type ApiRequest struct {
	token   string
	method  string
	payload interface{}
	url     *url.URL
}

func NewApiRequest(method, endpoint string, payload interface{}, params *url.Values, token string) (r ApiRequest, err error) {
	var (
		u *url.URL
	)
	method = strings.ToUpper(method)
	if method == "POST" && payload == nil {
		return r, fmt.Errorf("Payload empty")
	}
	if method == "GET" && payload != nil {
		return r, fmt.Errorf("Can't send payload via GET")
	}

	u, err = url.Parse(BaseURL + "/" + endpoint)
	if err != nil {
		log.Fatal(err)
	}

	if params != nil {
		u.RawQuery = params.Encode()
	}

	r = ApiRequest{
		url:     u,
		token:   token,
		method:  method,
		payload: payload,
	}

	return r, nil
}

func (r ApiRequest) URL() string {
	return r.url.String()
}

func (r ApiRequest) Send() (rawResponse []byte, err error) {
	var (
		req *http.Request
	)

	switch r.method {
	case http.MethodPost:
		body := new(bytes.Buffer)

		encoder := json.NewEncoder(body)
		if err := encoder.Encode(r.payload); err != nil {
			return rawResponse, err
		}

		req, err = http.NewRequest("POST", r.url.String(), body)
		if err != nil {
			return rawResponse, err
		}
	case http.MethodGet:
		req, err = http.NewRequest("GET", r.url.String(), nil)
		if err != nil {
			return rawResponse, err
		}
	default:
		return rawResponse, fmt.Errorf("Method is not recognised: %s", r.method)
	}

	req.Header.Set("Authorization", "Bearer "+r.token)
	//req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("User-Agent", "Feedservice")
	req.Header.Add("Content-Type", "application/json")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)

	defer cancel()

	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return rawResponse, err
	}
	//defer resp.Body.Close()

	// If request limit exceeded: recursive retry, 1 minute delay
	if resp.StatusCode == 429 {
		log.Println("Request limit exceeded, recursive retry")
		time.Sleep(time.Minute)
		return r.Send()
	}

	if resp.StatusCode != http.StatusOK {
		return rawResponse, fmt.Errorf("Request failed: %s - %s", resp.Status, req.URL.String())
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return rawResponse, err
	}

	if err != nil {
		return rawResponse, fmt.Errorf("Request failed: %v", err)
	}

	return b, nil
}
