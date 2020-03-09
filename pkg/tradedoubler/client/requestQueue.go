package tradedoublerclient

import (
	"errors"
	"fmt"
	"sync"
)

type requestQueue struct {
	connection *Connection
	queue      []Request
}

// PushGetRequest pushes a get request to the queue
func (rq *requestQueue) pushGetRequest(endpoint string) error {
	rq.queue = append(
		rq.queue,
		getRequest{
			Connection: rq.connection,
			Endpoint:   endpoint,
		},
	)
	return nil
}

// executeQueue pushes a get request to the queue
func (rq *requestQueue) execute() ([][]byte, error) {
	var responses [][]byte

	if len(rq.queue) < 1 {
		return responses, errors.New("No requests in the queue. Push to the queue first")
	}

	responses = make([][]byte, len(rq.queue))

	var wg sync.WaitGroup
	for i := range rq.queue {
		wg.Add(1)
		go func(idx int) {
			var err error
			responses[idx], err = rq.queue[idx].Send()
			if err != nil {
				fmt.Println(err)
			}
			wg.Done()
		}(i)

		// Wait for completion every 16 pending requests, to not flood the endpoint
		if i%16 == 0 {
			wg.Wait()
		}
	}
	wg.Wait()

	// clear queue and release to the garbage collector
	rq.queue = nil

	return responses, nil
}
