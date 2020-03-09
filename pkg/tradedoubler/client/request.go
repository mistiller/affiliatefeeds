package tradedoublerclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	MaxRetries = 3
)

// Request is an interface implemented by postRequest and getRequest
// It is used for the requestQueue to allow for async execution of generic requests
type Request interface {
	Send() ([]byte, error)
	getResponseError(rawResponse []byte) (string, error)
}

/* -----------------------------------------------
-- getRequest implements the Request interface  --
------------------------------------------------*/
type getRequest struct {
	Connection *Connection
	Endpoint   string
}

func (g getRequest) getResponseError(rawResponse []byte) (errorMessage string, err error) {
	var (
		errorReceiver map[string][]map[string]interface{}
	)

	err = json.Unmarshal(rawResponse, &errorReceiver)
	if err != nil {
		return errorMessage, err
	}

	errorMessage = errorReceiver["errors"][0]["message"].(string)

	return errorMessage, nil
}

// Send implements the Request Interface, returns raw bytes to be marshalled on a higher level
func (g getRequest) Send() ([]byte, error) {
	var rawResponse []byte
	url := "http://api.tradedoubler.com/1.0/" + g.Endpoint + fmt.Sprintf("?token=%s", g.Connection.token)

	// resend request up to 5 times if we don't get a 200 response
	// return on any other error
	var statusCode int
	for i := 0; i < MaxRetries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			return rawResponse, err
		}
		defer resp.Body.Close()

		rawResponse, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return rawResponse, err
		}

		statusCode = resp.StatusCode

		// break the loop if the request was successful
		if statusCode == http.StatusOK {
			break
		}
	}

	if statusCode != http.StatusOK {
		msg, err := g.getResponseError(rawResponse)
		if err != nil {
			msg = fmt.Sprintf("Failed to unmarshal response error message: /n %d: %s", statusCode, err)
		}
		return rawResponse, errors.New(msg)
	}

	return rawResponse, nil
}
