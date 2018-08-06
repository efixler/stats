package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"github.com/efixler/taxat/stats/stackdriver"
)

const (
	projectEnvId = "GOOGLE_CLOUD_PROJECT"
	kindTimeSeries = 1 << 0
	kindCounter = 1 << 1
)

var (
	help bool
 	project string
 	metric string
 	points []int
	usageTarget = os.Stderr
	sendFn func(context.Context, string, ...int) error
)



func main() {
	stackdriver.Sink.SetProjectId(project)
	ctx := context.Background()
	err := sendFn(ctx, metric, points...)
	if err != nil {
		croak(fmt.Sprintf("Error sending data to %s:\n\t%s", metric, err))
	}
	fmt.Println("Done")
}


func init() {
	flag.CommandLine.SetOutput(usageTarget)
	flag.CommandLine.Usage = Usage
	flag.BoolVar(&help, "h", false, "Display this help message")
	flag.StringVar(&project, "project", os.Getenv(projectEnvId), 
		fmt.Sprintf("GCP project name (default: %s environment variable)", projectEnvId))
	isTimeSeries := flag.Bool("t", false, "Send to a time series metric")
	isCounter := flag.Bool("c", false, "Send to a counter metric.")
	flag.Parse()
	if project == "" || len(flag.Args()) == 0 {
		croak("You must specify a project and some metrics to send")
	}
	kindOfMetric := 0
	if *isTimeSeries {
		kindOfMetric = kindOfMetric | kindTimeSeries
	}
	if *isCounter {
		kindOfMetric = kindOfMetric | kindCounter
	}
	switch kindOfMetric {
		case kindTimeSeries:	
			sendFn  = func(ctx context.Context, name string, data ...int) error {
				fmt.Printf("Timing timer %s...", name)
				err := stackdriver.Sink.WriteTimeSeries(ctx, name, data...)
				if err != nil {
					return err
				}
				return nil
			}
		case kindCounter: 
			sendFn  = func(ctx context.Context, name string, data ...int) error {
				fmt.Printf("Counting %s...", name)
				err := stackdriver.Sink.IncrementCounter(ctx, name, data...)
				if err != nil {
					return err
				}
				return nil
			}
		default:
			croak("You must specify a kind of metric")
	}	
	
	switch len(flag.Args()) {
		case 0: croak("Metric name not specified")
		case 1: croak("Please specify at least one data point")
		default:
			metric = flag.Args()[0]
			for _, arg := range flag.Args()[1:] {
				point, err := strconv.Atoi(arg)
				if err != nil {
					croak(fmt.Sprintf("Data point argument '%s' was not in numeric format", arg))
				}
				points = append(points, point)
			}		
	}
}

func croak(message string) {
	fmt.Fprintf(usageTarget, "\n*** Error: %s ***\n", message)
	Usage()
	os.Exit(2)
}

func Usage() {
	clean := path.Clean(os.Args[0])
	fmt.Fprintf(usageTarget, usage, clean, clean)
	flag.PrintDefaults()
	fmt.Fprintln(usageTarget, "")
}

var usage = `
%s: Push data points to stackdriver custom metrics

Usage:
-----
%s metric_name -c|t metric_name data_1 [data_2 ...]

Flags:
-----
`
