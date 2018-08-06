package stats

import (
	"context"
)

type Sink interface {
	WriteCounters(ctx context.Context, counters ...*Counter) error
	WriteTimers(ctx context.Context, timers ...*Timer) error
}
