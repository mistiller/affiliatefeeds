package awinclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const (
	DateFormat = "2006-01-02T15:04:05"
)

type Amount struct {
	Amount  string `json:"amount"`
	Curreny string `json:"currency"`
}

type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type TransactionPart struct {
	CommissionGroupId   uint64  `json:"commissionGroupId"`
	Amount              float64 `json:"amount"`
	CommissionAmount    float64 `json:"commissionAmount"`
	CommissionGroupCode string  `json:"commissionGroupCode"`
	CommissionGroupName string  `json:"commissionGroupName"`
}

type Transaction struct {
	ID                           uint64            `json:"id"`
	URL                          string            `json:"url"`
	AdvertiserID                 uint64            `json:"advertiserId"`
	PublisherID                  uint64            `json:"publisherId"`
	CommissionSharingPublisherId uint64            `json:"commissionSharingPublisherId"`
	SiteName                     string            `json:"siteName"`
	CommissionStatus             string            `json:"commissionStatus"`
	CommissionAmount             Amount            `json:"commissionAmount"`
	SaleAmount                   Amount            `json:"saleAmount"`
	IPHash                       int64             `json:"ipHash"`
	CustomerCountry              string            `json:"customerCountry"`
	ClickRefs                    map[string]string `json:"clickRefs"`
	ClickDate                    string            `json:"clickDate"`
	TransactionDate              string            `json:"transactionDate"`
	ValidationDate               string            `json:"validationDate"`
	Type                         string            `json:"type"`
	DeclineReason                string            `json:"declineReason"`
	VoucherCodeUsed              bool              `json:"voucherCodeUsed"`
	VoucherCode                  string            `json:"voucherCode"`
	LapseTime                    uint64            `json:"lapseTime"`
	Amended                      bool              `json:"amended"`
	AmendReason                  string            `json:"amendReason"`
	OldSaleAmount                Amount            `json:"oldSaleAmount"`
	OldCommissionAmount          Amount            `json:"oldCommissionAmount"`
	ClickDevice                  string            `json:"clickDevice"`
	TransactionDevice            string            `json:"transactionDevice"`
	PublisherURL                 string            `json:"publisherUrl"`
	AdvertiserCountry            string            `json:"advertiserCountry"`
	OrderRef                     string            `json:"orderRef"`
	CustomParameters             []Param           `json:"customParameters"`
	TransactionParts             []TransactionPart `json:"transactionParts"`
	PaidToPublisher              bool              `json:"paidToPublisher"`
	PaymentID                    int64             `json:"paymentId"`
	TransactionQueryID           int64             `json:"transactionQueryId"`
	originalSaleAmount           Amount            `json:"originalSaleAmount"`
}

type TransactionReport struct {
}

func GetTransactions(countryCode string, startDate, endDate time.Time, apiToken string) (transactions []Transaction, err error) {
	var (
		acc           AccountResponse
		parsed        []Transaction
		programmesReq ApiRequest
		progs         []Programme
	)

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 100*time.Millisecond)

	acc, err = GetAccounts(apiToken)
	if err != nil {
		return transactions, fmt.Errorf("Get accounts - %v", err)
	}

	start := startDate.Format(DateFormat)
	end := endDate.Format(DateFormat)
	if !endDate.After(startDate) {
		return transactions, fmt.Errorf("Date parameters inconsistent - end before start")
	}

	for i := range acc.Accounts {
		progs, err = GetPublisherProgrammes(acc.Accounts[i].AccountID, apiToken, countryCode)
		if err != nil {
			return transactions, fmt.Errorf("Get programmes - %v", err)
		}

		u := url.Values{}
		u.Add("accessToken", apiToken)
		u.Add("timezone", "UTC")
		u.Add("endDate", end)
		u.Add("startDate", start)
		for j := range progs {
			u.Add("advertiserId", fmt.Sprintf("%d", progs[j].ProgrammeInfo.ID))
		}

		programmesReq, err = NewApiRequest(
			"GET",
			fmt.Sprintf("publishers/%d/transactions/", acc.Accounts[i].AccountID),
			nil,
			&u,
			apiToken,
		)
		if err != nil {
			return transactions, fmt.Errorf("Build transaction request - %v", err)
		}

		resp, err := programmesReq.Send()
		if err != nil {
			return transactions, fmt.Errorf("Query trasnactions - %v", err)
		}

		err = json.Unmarshal(resp, &parsed)
		if err != nil {
			return transactions, fmt.Errorf("Parse transaction - %v", err)
		}

		for i := range parsed {
			transactions = append(transactions, parsed[i])
		}
	}

	return transactions, nil
}
