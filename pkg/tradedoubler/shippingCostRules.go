package tradedoubler

import "fmt"

type shippingPriceRule struct {
	FeedID        int32
	freeThreshold float32
	basePrice     float32
}

// ShippingCostRuleSet contains definitions for free shipping on a feed / store level
// Example "SEK": {999: &shippingPriceRule}
type ShippingCostRuleSet map[string]map[int32]*shippingPriceRule

// NewShippingCostRuleSet returns an initialized NewShippingPriceRuleSet
func NewShippingCostRuleSet() ShippingCostRuleSet {
	return make(ShippingCostRuleSet)
}

func (s ShippingCostRuleSet) Add(feedID int32, currency string, freeThreshold, basePrice float32) {
	var exists bool

	_, exists = s[currency]
	if !exists {
		s[currency] = make(map[int32]*shippingPriceRule)
	}
	s[currency][feedID] = &shippingPriceRule{
		FeedID:        feedID,
		freeThreshold: freeThreshold,
		basePrice:     basePrice,
	}
}

func (s ShippingCostRuleSet) GetPrice(currency string, feedID int32, sellingPrice float32) (price string, err error) {
	var exists bool

	_, exists = s[currency][feedID]
	if !exists {
		return "", fmt.Errorf("No rule for shipping costs - %s - %d", currency, feedID)
	}

	if sellingPrice >= s[currency][feedID].freeThreshold {
		return "0", nil
	}

	return fmt.Sprintf("%f", s[currency][feedID].basePrice), nil
}
