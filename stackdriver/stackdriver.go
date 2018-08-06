// This package is used to direct custom stats to stackdriver

package stackdriver 

import (
	"context"
	"errors"
	"strings"
	"time"
	"github.com/efixler/multierror"
	"github.com/efixler/taxat/stats"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/monitoring/v3"
)

const (
	typeNamePrefix = "custom.googleapis.com/"
	defaultWindowSeconds = int64(60 * 5)
)

var (
	NoData = errors.New("No data supplied for metric")
)

type sink struct {
	projectId		string
	windowSeconds 	int64
}

var Sink = &sink{
	projectId: "arctic-anvil-180318", //mark 
	windowSeconds: defaultWindowSeconds, 
} 

func (s *sink) ProjectId() string {
	return s.projectId
}

func (s *sink) ProjectResource() string {
	return "projects/" + s.projectId
}

func (s *sink) SetProjectId(pid string) {
	s.projectId = pid
}


// In the Stackdriver implementation, ".count" is always appended to the
// name of a counter, just to prevent naming clashes with timers.
func (s *sink) CreateCounter(ctx context.Context, name string) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	md := monitoring.MetricDescriptor{
		Type: fqTypeName(name) + ".count",
		MetricKind: "CUMULATIVE", //think it should be DELTA but that's not supported for custom
		ValueType: "INT64",
		Unit:	"1",
		Description: name + " counter",
		DisplayName: "Count of " + name,
	}
	_, err = client.Projects.MetricDescriptors.Create(s.ProjectResource(), &md).Do()
	if err != nil {
		return err
	}
	return nil
}

func CreateCounter(ctx context.Context, name string) error {
	return Sink.CreateCounter(ctx, name)
}

// If no data is provided here, it is assumed that the caller wants to increment the counter by 1.
// This method sends the data upstream immediately.
func (s *sink) IncrementCounter(ctx context.Context, name string, incr ...int) error {
	if len(incr) == 0 {
		incr = []int{1}
	}
	var val int64
	for _, bump := range incr {
		val += int64(bump)
	}
	if val == 0 {
		return nil
	}
	metric := &monitoring.Metric{
		Type: fqTypeName(name) + ".count", //todo: check for .count in name, don't double up
	}
	resource := &monitoring.MonitoredResource{
		Type: "global",
	}
	start, end := s.timeWindowBounds()
	interval := &monitoring.TimeInterval{
		StartTime: start.Format(time.RFC3339Nano),
		EndTime: end.Format(time.RFC3339Nano),
	}
	points := []*monitoring.Point{
		&monitoring.Point{
			Interval: interval,
			Value: &monitoring.TypedValue{Int64Value: &val},
		},
	}
	timeSeries := &monitoring.TimeSeries{
		Metric: metric,
		Resource: resource, 
		Points: points,
	}
	r := monitoring.CreateTimeSeriesRequest{
		TimeSeries: []*monitoring.TimeSeries{timeSeries},
	}
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.Projects.TimeSeries.Create(s.ProjectResource(), &r).Do()
	if err != nil {
		return err
	}
	return nil
}

func IncrementCounter(ctx context.Context, name string, incr ...int) error {
	return Sink.IncrementCounter(ctx, name, incr...)
}

func (s *sink) timeWindowBounds() (time.Time, time.Time) {
	t := time.Now().Unix()
	mod := t % s.windowSeconds
	tStart := t - mod
	tEnd := t + s.windowSeconds - mod
	return time.Unix(tStart, 0).UTC(), time.Unix(tEnd, 0).UTC()
}

func (s *sink) CreateTimeSeries(ctx context.Context, name string) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	md := monitoring.MetricDescriptor{
		Type: fqTypeName(name),
		MetricKind: "GAUGE",
		ValueType: "INT64",
		Unit:	"ms",
		Description: name + " time series",
		DisplayName: name + " time in milliseconds",
	}
	_, err = client.Projects.MetricDescriptors.Create(s.ProjectResource(), &md).Do()
	if err != nil {
		return err
	}
	return nil
}

func CreateTimeSeries(ctx context.Context, name string) error {
	return Sink.CreateTimeSeries(ctx, name)
}

// If multiple time values are supplied, they are averaged and sent as one data point. 
// This doesn't seem right, but stackdriver seems to only want one point per timeframe.
// 
// If no durations are passed, the method returns a NoData error. Any other errors returned
// indicate a Stackdriver API/service issue.
func (s *sink) WriteTimeSeries(ctx context.Context, name string, durationsMs ...int) error {
	if len(durationsMs) == 0 {
		return NoData
	}
	metric := &monitoring.Metric{
		Type: fqTypeName(name),
	}
	resource := &monitoring.MonitoredResource{
		Type: "global",
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	interval := &monitoring.TimeInterval{
		StartTime: now,
		EndTime: now,
	}
	var d64 int64
	switch len(durationsMs) {
		case 1:
			d64 = int64(durationsMs[0])
		default:
			for i:=0; i < len(durationsMs); i++ {
				durationsMs[0] = durationsMs[0] + durationsMs[i]
			}
			avg := float64(durationsMs[0])/float64(len(durationsMs))
			d64 = int64(avg) // (truncated)
	}
	points := []*monitoring.Point{
		&monitoring.Point{
			Interval: interval, 
			Value: &monitoring.TypedValue{
				Int64Value: &d64,
			},
		},
	}
	timeSeries := monitoring.TimeSeries{
		Metric: metric,
		Resource: resource, 
		Points: points, //don't know why this is called points when stackdriver will only accept 1
	}
	r := monitoring.CreateTimeSeriesRequest{
		TimeSeries: []*monitoring.TimeSeries{&timeSeries}, // could send multiple metric utilizing this
	}
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.Projects.TimeSeries.Create(s.ProjectResource(), &r).Do()
	if err != nil {
		return err
	}
	return nil
}

func WriteTimeSeries(ctx context.Context, name string, durationsMs ...int) error {
	return Sink.WriteTimeSeries(ctx,name, durationsMs...)
}


func fqTypeName(shortName string) string {
	if strings.Index(shortName, typeNamePrefix) == 0 {
		return shortName
	}
	return typeNamePrefix + shortName
}

func getClient(ctx context.Context) (*monitoring.Service, error) {
	hc, err := google.DefaultClient(ctx, monitoring.MonitoringScope)
	if err != nil {
		return nil, err
	}
	s, err := monitoring.New(hc)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Write all of the supplied counters to the data store. Implements the Sink interface 
func (ss *sink) WriteCounters(ctx context.Context, counters ...*stats.Counter) error {
	me := make(taxat.MultiError,0)
	for _, counter := range counters {
		if err := IncrementCounter(ctx, counter.Name(), counter.Data()); err != nil {
			me = append(me, err)
		}
	}
	if len(me) != 0 {
		return me
	}
	return nil
}

// Write all of the supplied timers to the data store. Implements the Sink interface 
func (ss *sink) WriteTimers(ctx context.Context, timers ...*stats.Timer) error {
	me := make(taxat.MultiError,0)
	for _, timer := range timers {
		if err := WriteTimeSeries(ctx, timer.Name(), timer.Milliseconds()); err != nil {
			me = append(me, err)
		}
	}
	if len(me) != 0 {
		return me
	}
	return nil
}


func (s *sink) DeleteMetric(ctx context.Context, name string) error {
	fqn := s.ProjectResource() + "/metricDescriptors/custom.googleapis.com/" + name  
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.Projects.MetricDescriptors.Delete(fqn).Do()
	return err
}

