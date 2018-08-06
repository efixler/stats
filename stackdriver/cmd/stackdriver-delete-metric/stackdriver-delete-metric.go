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
)

var help bool
var project string
var metrics []string
var usageTarget = os.Stderr


func main() {
	stackdriver.Sink.SetProjectId(project)
	ctx := context.Background()
	for _, metric := range metrics {
		if err := stackdriver.Sink.DeleteMetric(ctx, metric); err != nil {
			fmt.Printf("Error deleting metric %s:\n\t%s\n", metric, err)
		} else {
			fmt.Printf("Deleted metric %s\n", metric)
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
	flag.Parse()
	if project == "" {
		Usage()
		os.Exit(2)
	}
	if len(flag.Args()) == 0 {
		Usage()
		os.Exit(2)
	}
	metrics = flag.Args()
}

func Usage() {
	clean := path.Clean(os.Args[0])
	fmt.Fprintf(usageTarget, usage, clean, clean)
	flag.PrintDefaults()
	fmt.Fprintln(usageTarget, "")
}

var usage = `
%s: Delete custom stackdriver metrics

Usage:
-----
%s metric_name [metric_name metric_name ...]

Flags:
-----
`
