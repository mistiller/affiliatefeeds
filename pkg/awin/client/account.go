package awinclient

import (
	"encoding/json"
	"fmt"
)

type Account struct {
	AccountID   uint64 `json:"accountId"`
	AccountName string `json:""accountName"`
	AccountType string `json:""accountType"`
	UserRole    string `json:""userRole"`
}

type AccountResponse struct {
	UserID   uint64    `json:"userId`
	Accounts []Account `json:"accounts"`
}

func GetAccounts(apiToken string) (acc AccountResponse, err error) {
	accountsReq, err := NewApiRequest(
		"GET",
		"accounts",
		nil,
		nil,
		apiToken,
	)

	b, err := accountsReq.Send()
	if err != nil {
		return acc, fmt.Errorf("Get accounts - %v", err)
	}

	err = json.Unmarshal(b, &acc)
	if err != nil {
		return acc, fmt.Errorf("Unmarshal accounts - %v", err)
	}

	return acc, nil
}
