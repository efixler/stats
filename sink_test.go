package stats

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type dummySink struct {
	counterCount int
	timerCount   int
}

func (ds *dummySink) WriteCounters(ctx context.Context, counters ...*Counter) error {
	ds.counterCount = ds.counterCount + len(counters)
	return nil
}

func (ds *dummySink) WriteTimers(ctx context.Context, timers ...*Timer) error {
	ds.timerCount = ds.timerCount + len(timers)
	return nil
}

func TestDaemon(t *testing.T) {
	// se
	parentContext, cancelF := context.WithCancel(context.Background())
	rs := newRequestStats()
	sink := &dummySink{}
	requestContext := initRequestContext(parentContext, rs, sink)
	var i int
	for i = 0; i < 10; i++ {
		StartTimer(requestContext, "test_timer")
		Increment(requestContext, fmt.Sprintf("test_counter_%d", i))
		FinishTimer(requestContext, "test_timer")
	}
	cancelF()
	// NB: Following is to let the daemon finish, but baking in an
	// explicit notification channel would be better
	select {
	case <-time.After(100 * time.Millisecond):
	}
	if sink.counterCount != i {
		t.Errorf("Dropped counters: expected %d but only sent %d", i, sink.counterCount)
	}
	if sink.timerCount != i {
		t.Errorf("Dropped timers: expected %d but only sent %d", i, sink.timerCount)
	}

}
