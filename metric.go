// Stats package is used to collect in-request count and timing metrics.
// It's influenced by StatsD and provides a similar interface.
//
// Attach stats to a request via the router middleware functionality. The middleware
// APIs are developed and tested around gorilla mux, but should work with any middleware
// API that accepts http.Handler implementations.
//
// stats pushes data upstream using a background goroutine. Background processing is also
// supported in the Appengine Standard Environment, but in all scaling modes. See 
// https://cloud.google.com/appengine/docs/standard/go/modules/runtime#RunInBackground
// for more information about background activities and scaling modes in Appengine.
//
// Backends are pluggable. A metrics storage backend need only implement the Sink interface.
// There is a Sink implementation for Stackdriver at https://github.com/efixler/stats/stackdriver.
//
// See the examples for usage info and more details.
package stats

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	TimerNotStarted          = errors.New("Timer was never started")
	TimerNotFinished         = errors.New("Timer was never finished")
	RequestMetricsNotInitted = errors.New("Request metrics were not initialized (see stats.Metrics)")
	legalMetricName          = regexp.MustCompile(`^[a-z]+[\w./]+[a-zA-Z0-9]$`)
	IllegalMetricName        = errors.New(fmt.Sprintf("Names must match %s and not have consecutive dots or slashes", legalMetricName))
	NoSink                   = errors.New("No sink set up for storing metrics")
	NoSuchMetric             = errors.New("No metric by that name")
)

const (
	n2ms  int64 = 1000000
	msRnd       = n2ms / 2
)

type Metric interface {
	Name() string
	Data() int
}

type metric struct {
	name string
	data int
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Data() int { // this should maybe be an int64
	return m.data
}

type Counter struct {
	*metric
}

func (c *Counter) Increment() {
	c.data++
}

func (c *Counter) Decrement() {
	c.data--
}

type Timer struct {
	*metric
	startTime int64
}

// Nanoseconds
func (t *Timer) Duration() int64 {
	return int64(t.data)
}

func (t *Timer) Milliseconds() int {
	d64 := int64(t.data)
	msec := d64 / n2ms
	if d64%n2ms >= msRnd {
		msec++
	}
	return int(msec)
}

func (t *Timer) String() string {
	return fmt.Sprintf("T%s: %s", t.name, time.Duration(int64(t.data)))
}

func NewCounter(bucket string) (*Counter, error) {
	if err := checkMetricName(bucket); err != nil {
		return nil, err
	}
	c := &Counter{metric: &metric{name: bucket, data: 0}}
	return c, nil
}

func NewTimer(bucket string) (*Timer, error) {
	if err := checkMetricName(bucket); err != nil {
		return nil, err
	}
	t := &Timer{metric: &metric{name: bucket}, startTime: time.Now().UnixNano()}
	return t, nil
}

var ccds = regexp.MustCompile(`[./]{2,}?`)

func checkMetricName(n string) error {
	if !legalMetricName.MatchString(n) {
		return IllegalMetricName
	} else if ccds.MatchString(n) {
		return IllegalMetricName
	}
	return nil
}

func (t *Timer) Finish() error {
	if !t.Started() {
		return TimerNotStarted
	} else if !t.Finished() {
		now := time.Now().UnixNano()
		t.data = int(now - t.startTime)
	}
	return nil // second+ finish will noop
}

func (t *Timer) Started() bool {
	return t.startTime != 0
}

func (t *Timer) Finished() bool {
	return t.data != 0
}
