package awinclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type CommissionRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Type string  `json:"type"`
}

type CommissionGroup struct {
	GroupID    uint64  `json:"groupID"`
	GroupCode  string  `json:"groupCode"`
	GroupName  string  `json:"groupName"`
	Type       string  `json:"type"` // "percentage" vs. "fix"
	Percentage float64 `json:"percentage,omitempty"`
	Amount     float64 `json:"amount,omitempty"`
	Currency   string  `json:"currency"`
}

type CommissionsList struct {
	Advertiser       uint64            `json:"advertiser"`
	Publisher        uint64            `json:"publisher"`
	CommissionGroups []CommissionGroup `json:"commissionGroups"`
}

func GetCommissionGroups(publisherID, advertiserID uint64, apiToken string) (cg []CommissionGroup, err error) {
	var (
		r    ApiRequest
		list CommissionsList
	)
	r, err = NewApiRequest(
		"GET",
		fmt.Sprintf("publishers/%d/commissiongroups", publisherID),
		nil,
		&url.Values{
			"advertiserId": []string{fmt.Sprintf("%d", advertiserID)},
		},
		apiToken,
	)

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 100*time.Millisecond)

	b, err := r.Send()
	if err != nil {
		return cg, err
	}

	err = json.Unmarshal(b, &list)
	if err != nil {
		return cg, err
	}

	// Check whether every perecentage type has a percentage and every fix-type has an amount
	for i := range list.CommissionGroups {
		if (list.CommissionGroups[i].Type == "percentage" && list.CommissionGroups[i].Percentage == 0) ||
			(list.CommissionGroups[i].Type == "fix" && list.CommissionGroups[i].Amount == 0) {
			return cg, fmt.Errorf("Commission Group inconsistent")
		}
	}

	return list.CommissionGroups, nil
}
