package awinclient

import (
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Queue struct {
	requests map[uint64]Request
}

func NewQueue() (q *Queue) {
	return &Queue{
		requests: make(map[uint64]Request),
	}
}

func (q *Queue) Add(r Request) (err error) {
	var (
		h      hash.Hash64
		key    uint64
		exists bool
	)
	if q.requests == nil {
		q.requests = make(map[uint64]Request)
	}
	h = fnv.New64a()
	_, err = h.Write([]byte(r.URL()))
	if err != nil {
		return fmt.Errorf("Failed to build hashmap - %v", err)
	}
	key = h.Sum64()

	_, exists = q.requests[key]
	if !exists {
		q.requests[key] = r
	}

	return nil
}

func (q *Queue) Execute(strict bool) (rawResponse [][]byte, err error) {
	var (
		queueLength int

		wg sync.WaitGroup
	)

	queueLength = len(q.requests)
	if queueLength == 0 {
		return rawResponse, fmt.Errorf("Request Queue empty")
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	input := make(chan Request, queueLength)
	output := make(chan []byte, queueLength)

	// Increment waitgroup counter and create go routines
	for i := 0; i < ConcurrentRequests; i++ {
		wg.Add(1)
		go func(input chan Request, output chan []byte) {
			defer wg.Done()

			var resp []byte
			for req := range input {
				resp, err = req.Send()
				if err != nil {
					log.WithFields(
						log.Fields{
							"Target": req.URL(),
							"Error":  err,
						},
					).Debugln("Request error")
				}
				select {
				case <-ctx.Done():
					memLog("Request cancelled", mem, &maxMemory)
					output <- resp
				default:
					memLog("Request completed", mem, &maxMemory)
					if err != nil {
						log.Warnln(err)
					}
					output <- resp
				}
			}
		}(input, output)
	}

	// Producer: load up input channel with jobs
	for _, job := range q.requests {
		input <- job
	}

	log.WithField("Requests", queueLength).Info("Queue was scheduled")

	close(input)

	var i int
	for res := range output {
		i++
		if res == nil {
			continue
		}

		rawResponse = append(rawResponse, res)

		if i%10 == 0 || i == queueLength {
			progressBar("Execute Queue", i, queueLength)
			time.Sleep(50 * time.Millisecond)
		}

		select {
		case <-ctx.Done():
			close(output)
			cancel()
			log.Debugln("Go routine canceled")
			break
		default:
			if i >= queueLength {
				close(output)
				cancel()
				break
			}
		}
	}

	wg.Wait()

	q.requests = nil
	runtime.GC()

	return rawResponse, nil
}
