package stats

// This file contains the key public APIs for the stats package.

import (
	"context"
	"github.com/efixler/logger"
	"github.com/efixler/multierror"
	"net/http"
	"strings"
)

type statsContextKey string
type statsSinkKey string

var (
	requestStatsKey = statsContextKey("requestStats")
	sinkKey         = statsSinkKey("statsSink")
)

// This is the middleware call to set up metrics for a request, probably in conjunction with Gorilla mux,
// as in:
//		router.Use(Metrics(sink))
// where sink implements the Sink interface.
func Metrics(sink Sink) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(initRequestContext(r.Context(), newRequestStats(), sink))
			next.ServeHTTP(w, r)
		})
	}
}

func initRequestContext(ctx context.Context, rc *requestStats, sink Sink) context.Context {
	ctx = statsToContext(ctx, rc)
	if sink != nil {
		rc.eventChannel = openMetricsChannel(ctx, sink)
		ctx = context.WithValue(ctx, sinkKey, sink)
	}
	return ctx
}

// Time every http request on this server. In many environments, this will be superfluous, and is
// provided mainly for testing. If you use this, always make this setup call after Use(Metrics(sink))
func TimeRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tPath := strings.Join([]string{strings.Trim(r.URL.Path, "/"), r.Method}, ".")
		StartTimer(r.Context(), tPath)
		next.ServeHTTP(w, r)
	})
}

// Increment the counter with the named bucket. Counters can be incremented
// multiple times within a request. The counter will get flushed when the request
// is finished.
//
//
// Metric buckets are created on demand. Metric names can have alphanumeric characters, 
// slashes, underscores, and dots.
//
// 
//  func addUserHandler(w http.ResponseWriter, r *http.Request) {
//    ...
//    err := stats.Increment(r.Context(), "add_user")
//    ...
//  }
// Errors returned here will generally be IllegalMetricName or RequestMetricsNotInitted.
// Errors relating to the backend will not be reported here, as events 
func Increment(ctx context.Context, bucket string) error {
	ctxMetrics, ok := statsFromContext(ctx)
	if !ok {
		return RequestMetricsNotInitted
	}
	c, ok := ctxMetrics.counters[bucket]
	if !ok {
		var err error
		c, err = newCounter(bucket)
		if err != nil {
			return err
		}
		ctxMetrics.counters[bucket] = c //could consider a lock here, but in request scope contention seems unlikely
	}
	c.Increment()
	return nil
}

// Starts a timer with the named bucket. Named buckets are created on demand, and can contain alphanumeric
// characters, slashes, underscores, and dots.
//
//  func newUserHandler(w http.ResponseWriter, r *http.Request) {
//      timerName := "new_user_time"
//      err := stats.StartTimer(r.Context(), timerName)
//      //... do work in here
//      err := stats.FinishTimer(r.Context(), timerName)
//  }
//
// Errors returned here will generally be IllegalMetricName or RequestMetricsNotInitted.
func StartTimer(ctx context.Context, bucket string) error {
	ctxMetrics, ok := statsFromContext(ctx)
	if !ok {
		return RequestMetricsNotInitted
	}
	if t, err := newTimer(bucket); err != nil {
		return err
	} else {
		ctxMetrics.timers[bucket] = t
		return nil
	}

}

// Finish the timer specified by bucket. 
// The finished  timer will be forwarded to the Sink, if one has been set up.
func FinishTimer(ctx context.Context, bucket string) error {
	ctxMetrics, ok := statsFromContext(ctx)
	if !ok {
		return RequestMetricsNotInitted
	}
	t, ok := ctxMetrics.timers[bucket]
	if !ok {
		return TimerNotStarted
	}
	err := t.Finish()
	if err != nil {
		return err
	}
	if err := ctxMetrics.sendTimer(bucket); err != nil {
		logger.Context.Warningf(ctx, "Error pushing finished timer %s into event stream: %s", bucket, err)
	}
	return nil
}

////// end of public APIs

// flushAll will ensure that all timers are finished and then send them on.
// In-progress errors do not stop execution. They are collected and returned in the error
// (which is a MultiError)
func flushAll(ctx context.Context) error {
	me := make(multierror.MultiError, 0)
	// just in case.

	ctxMetrics, ok := statsFromContext(ctx)
	if !ok {
		return RequestMetricsNotInitted
	}
	for _, timer := range ctxMetrics.timers {
		if err := timer.Finish(); err != nil {
			me = append(me, err)
		}
	}
	if err := ctxMetrics.sendAll(); err != nil {
		me = append(me, err)
	}
	return me.NilWhenEmpty()
}

// Using a struct to store all the transient stats in the request context
// Current design has a constraint of 1 instance of a particular bucket/in a request.
// This doesn't matter for counters (which can always be incremented) but it does mean
// that, for timers, you can't have overlapping started timers in the same bucket (which
// probably indicates shitty code anyway)

type requestStats struct {
	counters     map[string]*Counter
	timers       map[string]*Timer
	eventChannel chan<- Metric
}

func statsToContext(ctx context.Context, rs *requestStats) context.Context {
	ctx = context.WithValue(ctx, requestStatsKey, rs)
	return ctx
}

func statsFromContext(ctx context.Context) (*requestStats, bool) {
	ctxMetrics, ok := ctx.Value(requestStatsKey).(*requestStats)
	return ctxMetrics, ok
}

func newRequestStats() *requestStats {
	rc := &requestStats{
		counters: make(map[string]*Counter),
		timers:   make(map[string]*Timer),
	}
	return rc
}

// Send the counter in the requested bucket upstream, and delete it from
// the map. If the counter's data == 0, don't bother sending it, since it's a noop,
// data-wise.
func (rs *requestStats) sendCounter(bucket string) error {
	c, ok := rs.counters[bucket]
	if !ok {
		return NoSuchMetric
	} else if c.Data() == 0 {
		return nil
	}
	defer delete(rs.counters, bucket)
	return rs._send(c)
}

// Send the timer in the requested bucket upstream, and delete it from
// the map. If the timer hasn't been finished, returns TimerNotFinished
func (rs *requestStats) sendTimer(bucket string) error {
	t, ok := rs.timers[bucket]
	if !ok {
		return NoSuchMetric
	} else if !t.Finished() {
		return TimerNotFinished
	}
	defer delete(rs.timers, bucket)
	return rs._send(t)
}

// Following marked _ because it should only be used inside
// request stats. This method only pushes events into the channel; it does not
// delete the metrics from the bucket store
func (rs *requestStats) _send(m Metric) error {
	if rs.eventChannel == nil {
		return NoSink
	}
	rs.eventChannel <- m
	return nil
}

// Send all metrics in the struct. Timers will not be sent if they aren't finished,
// and counters won't be sent if they haven't counted anything. These behaviors mirror
// the behaviors of the `sendTimer()` and `sendCounter()` one-offs.
func (rs *requestStats) sendAll() error {
	if rs.eventChannel == nil {
		return NoSink
	}
	me := make(multierror.MultiError, 0)
	for _, timer := range rs.timers {
		if !timer.Finished() {
			me = append(me, TimerNotFinished)
			continue
		}
		rs.eventChannel <- timer
	}
	rs.timers = nil // this struct is now b0rked
	for _, counter := range rs.counters {
		if counter.Data() == 0 {
			continue //not considering this an error. Zeroes are possible.
		}
		rs.eventChannel <- counter
	}
	rs.counters = nil
	return me.NilWhenEmpty()
}
