package storefront

import (
	"sync/atomic"
)

type IDCounter struct {
	ID *uint64
}

func NewCounter() (c *IDCounter) {
	return &IDCounter{
		ID: new(uint64),
	}
}

func (c *IDCounter) Increment(count uint64) {
	atomic.AddUint64(c.ID, count)
}

func (c *IDCounter) Get() uint64 {
	return *c.ID
}
