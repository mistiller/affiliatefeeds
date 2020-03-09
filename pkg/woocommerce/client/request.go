package wooclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Request is implemented for Batch/Post and Get
type Request interface {
	Send(w *Client) ([]byte, error)
}

// PostRequest can be used for synchronous requests to the products, attributes, or categories endpoint
type PostRequest struct {
	Endpoint string
	Locale   string
	Payload  Item
}

// Send implements the WooRequest interface
func (p PostRequest) Send(w *Client) ([]byte, error) {
	if w.initialized == false {
		return nil, fmt.Errorf("Please initialize with your credentials first. WooConnection.Init()")
	}

	//resp, err := w.client.Post(p.Endpoint+"?"+query.Encode(), p.Payload)

	resp, err := w.request(
		"POST",
		p.Endpoint,
		url.Values{
			"lang": []string{
				p.Locale,
			},
		},
		p.Payload,
	)
	if err != nil {
		return nil, fmt.Errorf("%v - %s", p.Endpoint, err)
	}
	defer resp.Close()

	body, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, err
	}

	err = checkResult(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// BatchPostRequest sends a payload of batch creations, updates and/or deletions
type BatchPostRequest struct {
	Endpoint string `json:"-"`
	Locale   string `json:"lang,omitempty"`
	Create   []Item `json:"create,omitempty"` // Create requests must not have IDs -the WC backend will generate them
	Update   []Item `json:"update,omitempty"` // Update requests must have IDs
	Delete   []int  `json:"delete,omitempty"` // Delete requests can only be IDs
}

// Send implements the Request Interface
func (b BatchPostRequest) Send(w *Client) ([]byte, error) {
	if w.initialized == false {
		return nil, errors.New("Please initialize with your credentials first. WooConnection.Init()")
	}

	resp, err := w.request(
		"POST",
		b.Endpoint,
		url.Values{
			"lang": []string{
				b.Locale,
			},
		},
		b,
	)
	if err != nil {
		return nil, fmt.Errorf("%v - %s", b.Endpoint, err)
	}
	defer resp.Close()

	body, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, err
	}

	err = checkResult(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// GetRequest implements GET request via a Client
type GetRequest struct {
	Endpoint string
	Params   url.Values
}

// Send implementes the Request interface
func (g GetRequest) Send(w *Client) (body []byte, err error) {
	if w.initialized == false {
		return nil, fmt.Errorf("Please initialize with your credentials first. Client.Init()")
	}

	//resp, err := w.client.Get(g.Endpoint, g.Params)
	resp, err := w.request("GET", g.Endpoint, g.Params, nil)
	if err != nil {
		return nil, fmt.Errorf("%v - %s", g.Endpoint, err)
	}
	defer resp.Close()

	body, err = ioutil.ReadAll(resp)
	if err != nil {
		return nil, err
	}

	err = checkResult(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (w *Client) request(method, endpoint string, params url.Values, data interface{}) (rc io.ReadCloser, err error) {
	urlstr := w.storeURL.String() + endpoint
	if params == nil {
		params = make(url.Values)
	}
	symb := "?"
	if strings.Contains(urlstr, "?") {
		symb = "&"
	}
	if w.storeURL.Scheme == "https" {
		urlstr += symb + w.basicAuth(params)
	} else {
		urlstr += symb + w.oauth(method, urlstr, params)
	}
	switch method {
	case http.MethodPost, http.MethodPut:
	case http.MethodDelete, http.MethodGet, http.MethodOptions:
	default:
		return rc, fmt.Errorf("Method is not recognised: %s", method)
	}

	body := new(bytes.Buffer)
	encoder := json.NewEncoder(body)
	if err := encoder.Encode(data); err != nil {
		return rc, err
	}

	req, err := http.NewRequest(method, urlstr, body)
	if err != nil {
		return rc, err
	}
	req.SetBasicAuth(w.key, w.secret)
	req.Header.Set("Content-Type", "application/json")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)

	req = req.WithContext(ctx)

	url := req.URL.String()

	resp, err := w.rawClient.Do(req)
	if err != nil {
		cancel()
		return rc, err
	}
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusCreated {

		cancel()
		return rc, fmt.Errorf("Request failed: %s - %s", resp.Status, url)
	}

	return resp.Body, nil
}

func (w *Client) basicAuth(params url.Values) string {
	params.Add("consumer_key", w.key)
	params.Add("consumer_secret", w.secret)
	return params.Encode()
}

func (w *Client) oauth(method, urlStr string, params url.Values) string {
	if w.OauthTimestamp.IsZero() {
		w.OauthTimestamp = time.Now()
	}
	params.Add("oauth_consumer_key", w.key)
	params.Add("oauth_timestamp", strconv.Itoa(int(w.OauthTimestamp.Unix())))
	nonce := make([]byte, 16)
	rand.Read(nonce)
	sha1Nonce := fmt.Sprintf("%x", sha1.Sum(nonce))
	params.Add("oauth_nonce", sha1Nonce)
	params.Add("oauth_signature_method", HashAlgorithm)
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var paramStrs []string
	for _, key := range keys {
		paramStrs = append(paramStrs, fmt.Sprintf("%s=%s", key, params.Get(key)))
	}
	paramStr := strings.Join(paramStrs, "&")
	params.Add("oauth_signature", w.oauthSign(method, urlStr, paramStr))
	return params.Encode()
}

func (w *Client) oauthSign(method, endpoint, params string) string {
	signingKey := w.secret
	if w.Version == "v3" {
		signingKey = signingKey + "&"
	}

	a := strings.Join([]string{method, url.QueryEscape(endpoint), url.QueryEscape(params)}, "&")
	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(a))
	signatureBytes := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signatureBytes)
}
