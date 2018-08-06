package stats

// Functions herein are to facilitate the asynchronous collection and dispatch of
// stats events.

// The operating theory is straightforward:
//
// The middleware opens a channel for stats events and launches a goroutine to drain them.
// the goroutine terminates when its context is cancelled (or when the events channel is closed).
// In appengine, the context is cancelled soon after the request finishes.

import (
	"context"
	"github.com/efixler/logger"
)

func openMetricsChannel(ctxo context.Context, sink Sink) chan<- Metric {
	events := make(chan Metric)
	waitForRequestDone(ctxo, events)

	runFunc := func(ctx context.Context) {
		for {
			select {
			case event := <-events:
				switch event.(type) {
				// todo: batch processing for the following 2 cases
				case *Counter:
					err := sink.WriteCounters(ctx, event.(*Counter))
					if err != nil {
						logger.Context.Errorf(ctx, "Error flushing counter: %s", err)
					}
				case *Timer:
					err := sink.WriteTimers(ctx, event.(*Timer))
					if err != nil {
						logger.Context.Errorf(ctx, "Error flushing timer: %s", err)
					}
				case nil:
					// when the channel is closed, we will see a nil value here
					return
				default:
					// exit if unexpected stuff happens, so we don't leak
					logger.Context.Errorf(ctx, "Unexpected event (type %T) passed to stat sink channel", event)
					return

				}
			case <-ctx.Done():
				// .Done() channel cannot be relied upon for background operations.
				return
			}
		}
	}
	err := startEventsListener(ctxo, runFunc)
	if err != nil {
		logger.Context.Errorf(ctxo, "Can't start metrics listener, events will not be flushed: %s", err)
	}
	return events
}

func waitForRequestDone(ctx context.Context, evtChan chan<- Metric) {
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			flushAll(ctx)
			defer close(evtChan)
		}
	}(ctx)
}
