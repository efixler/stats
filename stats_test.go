package stats

import (
	"context"
	"testing"
)

func TestMetricsInitted(t *testing.T) {
	ctx := context.Background()
	bucket := "my/test/metric"
	if err := Increment(ctx, bucket); err != RequestMetricsNotInitted {
		t.Errorf("Expected error %s, got %v", RequestMetricsNotInitted, err)
	}
	ctx = requestContextUsingMetrics()
	if err := Increment(ctx, bucket); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

var nameChecks = []struct {
	name string
	err  error
}{
	{"my/test/metric", nil},
	{"", IllegalMetricName},
	{"with/double//slashes.name", IllegalMetricName},
	{"ends/with/.", IllegalMetricName},
	{"illegal/char&", IllegalMetricName},
}

func TestNames(t *testing.T) {
	ctx := requestContextUsingMetrics()
	for _, test := range nameChecks {
		err := Increment(ctx, test.name)
		if err != test.err {
			t.Errorf("Test metric name %s: expected %v, got %v", test.name, test.err, err)
			continue
		}
	}
}

func requestContextUsingMetrics() context.Context {
	// just to make the example pretty
	ctx := context.Background()
	ctx = statsToContext(ctx, newRequestStats())
	return ctx
}
