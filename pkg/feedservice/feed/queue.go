package feed

import (
	"context"
	"fmt"

	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

const (
	// MaxConcurrentRequests defines how many feeds are processed simultaneously
	MaxConcurrentRequests = 8
)

//Queue allows to process multiple feeds at once
type Queue struct {
	queue          []Feed
	productionFlag bool
}

// NewQueueFromFeeds takes a slice of of the feed interfaces, returns pointer to Queue
func NewQueueFromFeeds(f []Feed, productionFlag bool) (q *Queue) {
	q = &Queue{
		productionFlag: productionFlag,
	}
	for i := range f {
		q.queue = append(q.queue, f[i])
	}
	return q
}

// AppendOne feed to queue so it can be processed later
func (q *Queue) AppendOne(f Feed) {
	q.queue = append(q.queue, f)
}

// AppendMany feeds to queue so it can be processed later
func (q *Queue) AppendMany(f []Feed) {
	for i := range f {
		q.queue = append(q.queue, f[i])
	}
}

// GetPM processes the queue of feeds and returns a deduplicated product map
func (q *Queue) GetPM(strict bool) (productMap *ProductMap, err error) {
	nsources := len(q.queue)
	if nsources < 1 {
		return productMap, fmt.Errorf("Empty queue")
	}
	var wg sync.WaitGroup

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	input := make(chan Feed, nsources)
	output := make(chan []Product, nsources)

	var errs uint64
	for i := 0; i < MaxConcurrentRequests; i++ {
		wg.Add(1)
		go func(input chan Feed, output chan []Product) {
			defer wg.Done()

			for f := range input {
				products, err := f.Get(q.productionFlag)
				if err != nil {
					log.WithField("Error", err).Warningln("Failed to download feed from queue")
					atomic.AddUint64(&errs, 1)
					output <- []Product{}
					continue
				}
				select {
				case <-ctx.Done():
					output <- []Product{}
					log.Println("Request cancelled")
				default:
					output <- products
				}
				products = nil
			}
		}(input, output)
	}

	// Producer: load up input channel with jobs
	for _, job := range q.queue {
		input <- job
	}
	log.WithField("Sources", nsources).Infoln("Queue prepared")

	close(input)

	var (
		products []Product
		counter  int
	)
	for res := range output {
		if errs > 0 {
			return productMap, fmt.Errorf("Error in feed queue")
		}
		for i := range res {
			if res[i].Name == "" {
				continue
			}
			products = append(products, res[i])
			if i%10000 == 0 || i == len(res) {
				log.WithFields(
					log.Fields{
						"Completed": i + 1,
						"Total":     len(res),
					},
				).Infoln("Receiving")
			}
		}
		counter++

		select {
		case <-ctx.Done():
			close(output)
			cancel()
			log.Debugln("Go routine canceled")
			break
		default:
			if counter >= nsources {
				close(output)
				cancel()
				log.Debugln("Go routine finished")
				break
			}
		}
	}

	wg.Wait()

	if len(products) == 0 {
		return productMap, fmt.Errorf("No products loaded from the queue")
	}

	productMap, err = PMFromSlice(products)
	if err != nil {
		return productMap, fmt.Errorf("Generating Product Map - %v", err)
	}

	return productMap, nil
}
