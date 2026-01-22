package async

import (
	"context"
	"sync"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/sink"
)

// AsyncSink wraps a Sink and processes trails asynchronously
type AsyncSink struct {
	sink       sink.Sink
	queue      chan *gotrails.Trail
	wg         sync.WaitGroup
	closed     bool
	closeMu    sync.Mutex
	workers    int
	onError    func(error)
	dropOnFull bool
}

// AsyncOption is an option for AsyncSink
type AsyncOption func(*AsyncSink)

// WithWorkers sets the number of worker goroutines
func WithWorkers(n int) AsyncOption {
	return func(a *AsyncSink) {
		if n > 0 {
			a.workers = n
		}
	}
}

// WithOnError sets the error handler
func WithOnError(fn func(error)) AsyncOption {
	return func(a *AsyncSink) {
		a.onError = fn
	}
}

// WithDropOnFull drops trails when the queue is full instead of blocking
func WithDropOnFull(drop bool) AsyncOption {
	return func(a *AsyncSink) {
		a.dropOnFull = drop
	}
}

// NewAsyncSink creates a new AsyncSink
func NewAsyncSink(s sink.Sink, queueSize int, opts ...AsyncOption) *AsyncSink {
	if queueSize <= 0 {
		queueSize = 1000
	}

	async := &AsyncSink{
		sink:    s,
		queue:   make(chan *gotrails.Trail, queueSize),
		workers: 1,
	}

	for _, opt := range opts {
		opt(async)
	}

	// Start workers
	for i := 0; i < async.workers; i++ {
		async.wg.Add(1)
		go async.worker()
	}

	return async
}

// worker processes trails from the queue
func (a *AsyncSink) worker() {
	defer a.wg.Done()

	for trail := range a.queue {
		if err := a.sink.Write(context.Background(), trail); err != nil {
			if a.onError != nil {
				a.onError(err)
			}
		}
	}
}

// Write queues a trail for async processing
func (a *AsyncSink) Write(ctx context.Context, trail *gotrails.Trail) error {
	a.closeMu.Lock()
	if a.closed {
		a.closeMu.Unlock()
		return nil
	}
	a.closeMu.Unlock()

	// Clone the trail to avoid race conditions
	cloned := trail.Clone()

	if a.dropOnFull {
		select {
		case a.queue <- cloned:
		default:
			// Queue full, drop the trail
		}
	} else {
		select {
		case a.queue <- cloned:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// Close closes the async sink and waits for all workers to finish
func (a *AsyncSink) Close() error {
	a.closeMu.Lock()
	if a.closed {
		a.closeMu.Unlock()
		return nil
	}
	a.closed = true
	a.closeMu.Unlock()

	close(a.queue)
	a.wg.Wait()

	return a.sink.Close()
}

// Name returns the name of the async sink
func (a *AsyncSink) Name() string {
	return "async:" + a.sink.Name()
}

// QueueLength returns the current queue length
func (a *AsyncSink) QueueLength() int {
	return len(a.queue)
}

// QueueCapacity returns the queue capacity
func (a *AsyncSink) QueueCapacity() int {
	return cap(a.queue)
}
