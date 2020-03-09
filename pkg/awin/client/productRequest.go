package awinclient

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

const toMeg uint64 = 1048576

type ProductRequest struct {
	url     string
	token   string
	maxRows int
}

func NewProductRequest(url, token string, maxRows int) (request *ProductRequest) {
	return &ProductRequest{
		url:     url,
		token:   token,
		maxRows: maxRows,
	}
}

func (r ProductRequest) URL() string {
	return r.url
}

func (r ProductRequest) Send() (b []byte, err error) {
	type Row map[string]*string

	var (
		header []string
		record []string
		row    Row
		rows   []Row
	)

	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		return b, err
	}
	req.Header.Add("Accept-Encoding", "gzip, deflate")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)

	defer cancel()

	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return b, err
	}
	defer resp.Body.Close()

	memLog("Starting to download Awin File", mem, &maxMemory)

	// If request limit exceeded: recursive retry
	if resp.StatusCode == 429 {
		log.Println("Request limit exceeded, recursive retry")
		time.Sleep(time.Minute)
		return r.Send()
	}

	if resp.StatusCode != http.StatusOK {
		return b, fmt.Errorf("Request failed: %s - %s", resp.Status, req.URL.String())
	}

	zipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return b, err
	}
	defer zipReader.Close()

	reader := csv.NewReader(zipReader)
	reader.LazyQuotes = true

	i := 0
	for {
		// read one row from csv
		record, err = reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return b, err
		}

		if i == 0 {
			header = record
			i++
			continue
		}

		if i%10000 == 0 {
			memLog(fmt.Sprintf("Downloaded %d rows", i), mem, &maxMemory)
		}

		row = make(Row, len(header))
		for k := range header {
			row[header[k]] = &record[k]
		}

		rows = append(rows, row)

		i++

		if r.maxRows > 0 && i >= r.maxRows {
			break
		}
	}

	reader = nil
	runtime.GC()

	b, err = json.Marshal(rows)
	if err != nil {
		return b, err
	}
	rows = nil

	memLog("Finished collecting", mem, &maxMemory)

	return b, nil
}
