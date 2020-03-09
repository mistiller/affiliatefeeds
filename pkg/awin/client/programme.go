package awinclient

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Domain struct {
	Domain string `json:"domain"`
}
type Region struct {
	Name   string `json:"name`
	Region string `json:"region`
}

type ProgrammeInfo struct {
	DisplayUrl      string   `json:"displayUrl"`
	ClickThroughUrl string   `json:"clickThroughUrl"`
	Name            string   `json:"name"`
	ID              uint64   `json:"id"`
	LogoUrl         string   `json:"logoUrl"`
	CurrencyCode    string   `json:"currencyCode"`
	PrimaryRegion   Region   `json:"primaryRegion"`
	ValidDomains    []Domain `json:"validDomains,omitempty"`
}

type KPI struct {
	AveragePaymentTime string  `json:"averagePaymentTime"`
	ApprovalPercentage float64 `json:"approvalPercentage"`
	EPC                float64 `json:"epc"`
	ConversionRate     float64 `json:"conversionRate"`
	ValidationDays     int     `json:"validationDays"`
	AwinIndex          float64 `json:"awinIndex"`
}

// Programme Details endpoint: https://wiki.awin.com/index.php/API_get_programmedetails
type Programme struct {
	ProgrammeInfo    ProgrammeInfo     `json:"programmeInfo"`
	KPI              KPI               `json:"kpi"`
	CommissionRange  []CommissionRange `json:"commissionRange"`
	CommissionGroups []CommissionGroup
}

func GetProgrammes(countryCode string, apiToken string) (outProgs []Programme, err error) {
	var (
		acc    AccountResponse
		progs  []Programme
		exists bool
	)

	acc, err = GetAccounts(apiToken)
	if err != nil {
		return outProgs, fmt.Errorf("Get accounts - %v", err)
	}

	activeIDMap := make(map[uint64]struct{})
	for i := range acc.Accounts {
		progs, err = GetPublisherProgrammes(acc.Accounts[i].AccountID, apiToken, countryCode)
		if err != nil {
			return outProgs, fmt.Errorf("Get programmes - %v", err)
		}
		for j := range progs {
			_, exists = activeIDMap[progs[j].ProgrammeInfo.ID]
			if !exists {
				progs[j].CommissionGroups, err = GetCommissionGroups(
					acc.Accounts[i].AccountID,
					progs[j].ProgrammeInfo.ID,
					apiToken,
				)
				if err != nil {
					return outProgs, fmt.Errorf("Get commissions - %v", err)
				}
				outProgs = append(outProgs, progs[j])
				activeIDMap[progs[j].ProgrammeInfo.ID] = struct{}{}
			}
		}
	}

	return outProgs, nil
}

func GetPublisherProgrammes(pubID uint64, apiToken, countryCode string) (progs []Programme, err error) {
	programmesReq, err := NewApiRequest(
		"GET",
		fmt.Sprintf("publishers/%d/programmes", pubID),
		nil,
		&url.Values{
			"relationship": []string{"joined"},
			"countryCode":  []string{countryCode},
		},
		apiToken,
	)

	b, err := programmesReq.Send()
	if err != nil {
		return progs, fmt.Errorf("Get programmes - %v", err)
	}

	var list []ProgrammeInfo
	err = json.Unmarshal(b, &list)
	if err != nil {
		return progs, fmt.Errorf("Unmarshal programmes - %v", err)
	}

	progs = make([]Programme, len(list))
	for i := range list {
		programmesReq, err := NewApiRequest(
			"GET",
			fmt.Sprintf("publishers/%d/programmedetails", pubID),
			nil,
			&url.Values{
				"advertiserId": []string{fmt.Sprintf("%d", list[i].ID)},
			},
			apiToken,
		)
		b, err := programmesReq.Send()
		if err != nil {
			return progs, fmt.Errorf("Get programmes - %v", err)
		}

		prog := Programme{}
		err = json.Unmarshal(b, &prog)
		if err != nil {
			return progs, fmt.Errorf("Unmarshal programmes - %v", err)
		}

		progs[i] = prog
	}

	return progs, nil
}
