package async

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/example/car-mall-intelligent-agent/internal/model"
)

type PricingProcessor interface {
	ProcessPricing(task model.PricingTask) model.PricingResult
}

type InMemoryQueue struct {
	ch chan model.PricingTask
	wg sync.WaitGroup
}

func NewInMemoryQueue(buffer int) *InMemoryQueue {
	return &InMemoryQueue{ch: make(chan model.PricingTask, buffer)}
}

func (q *InMemoryQueue) Publish(task model.PricingTask) {
	q.ch <- task
}

func (q *InMemoryQueue) RunConsumer(
	ctx context.Context,
	processor PricingProcessor,
	onProcessed func(task model.PricingTask, result model.PricingResult, cost time.Duration),
) {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-q.ch:
				start := time.Now()
				result := processor.ProcessPricing(task)
				if onProcessed != nil {
					onProcessed(task, result, time.Since(start))
				}
				log.Printf("[pricing-consumer] task=%s final_price=%.2f cost=%s",
					result.TaskID, result.FinalPrice, time.Since(start))
			}
		}
	}()
}

func (q *InMemoryQueue) Wait() {
	q.wg.Wait()
}
