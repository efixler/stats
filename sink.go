package stats

import (
	"context"
)

// Implement the Sink interface to connect a metrics storage backend to
// the stats package. See the stackdriver package for a sample implementation.
//
// The Sink methods accept multiple metrics variadically, to let the implementation
// optimize batching for its data store. A failure on one metric need not fail the entire batch --
// use http://github.com/efixler/multierror to return multiple errors to the caller, which
// will log them.
type Sink interface {
	WriteCounters(ctx context.Context, counters ...*Counter) error
	WriteTimers(ctx context.Context, timers ...*Timer) error
}
