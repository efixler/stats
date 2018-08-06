package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"github.com/efixler/taxat/stats/stackdriver"
)

const (
	projectEnvId = "GOOGLE_CLOUD_PROJECT"
	kindTimeSeries = 1 << 0
	kindCounter = 1 << 1
)

var help bool
var project string
var metrics []string
var usageTarget = os.Stderr
var createFn func(context.Context, string) error


func main() {
	stackdriver.Sink.SetProjectId(project)
	ctx := context.Background()
	for _, metric := range metrics {
		if err := createFn(ctx, metric); err != nil {
			croak(fmt.Sprintf("Error creating metric %s:\n\t%s\n", metric, err))
		} else {
			fmt.Printf("Created metric %s\n", metric)
		}
}
	os.Exit(0)
}


func init() {
	flag.CommandLine.SetOutput(usageTarget)
	flag.CommandLine.Usage = Usage
	flag.BoolVar(&help, "h", false, "Display this help message")
	flag.StringVar(&project, "project", os.Getenv(projectEnvId), 
		fmt.Sprintf("GCP project name (default: %s environment variable)", projectEnvId))
	isTimeSeries := flag.Bool("t", false, "Create a time series metric")
	isCounter := flag.Bool("c", false, "Create a counter metric. Will append '.count' to name")
	
	flag.Parse()
	if project == "" || len(flag.Args()) == 0 {
		croak("You must specify a project and some metrics to make")
	}
	metrics = flag.Args()
	kindOfMetric := 0
	if *isTimeSeries {
		kindOfMetric = kindOfMetric | kindTimeSeries
	}
	if *isCounter {
		kindOfMetric = kindOfMetric | kindCounter
	}
	switch kindOfMetric {
		case kindTimeSeries:	
			createFn  = func(ctx context.Context, name string) error {
				fmt.Printf("Creating timer %s\n", name)
				err :=  stackdriver.Sink.CreateTimeSeries(ctx, name)
				if err != nil {
					return err
				} // Things get weird if you don't send data right away
				err = stackdriver.Sink.WriteTimeSeries(ctx, name, 0)
				if err != nil {
					return err
				}
				return nil
			}
		case kindCounter: 
			createFn  = func(ctx context.Context, name string) error {
				fmt.Printf("Creating counter %s\n", name)
				err := stackdriver.Sink.CreateCounter(ctx, name)
				if err != nil {
					return err
				}
				err = stackdriver.Sink.IncrementCounter(ctx, name, 1)
				if err != nil {
					return err
				}
				return nil
			}
		default:
			croak("You must specify a kind of metric")
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
%s: Create custom stackdriver metrics

Usage:
-----
%s metric_name -c|t [metric_name metric_name ...]

Flags:
-----
`
