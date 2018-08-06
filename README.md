# stats
In-request counters and timers for golang web services

[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[godocs]: https://godoc.org/github.com/efixler/stats

stats is designed as a performant and easy-to-use stats collector, along the lines of StatsD. Backends are pluggable by implementing the 2-method sink interface.

The stats package has been used to some degree, but hasn't been battle-tested at high scale. Field reports and improvements are welcome.

This package may be useful for guidance on a handful of topics, such as:

* Using channels and goroutines to kick off parallel processing inside an http.Request without blocking the user response.
* Writing packages that compile and run inside the Appengine Standard Environment, and also compile and run outside the Appengine Standard Environment.
* The stackdriver subpackage demonstrates usage of Stackdriver to store user-defined metrics.

## Installation

`go get github.com/efixler/stats`

Installing stats will also install the Stackdriver backend and command-line helpers. See the stackdriver/ subfolder 
for details.

## Usage
The usage sample below assumes Stackdriver as the storage backend for metrics data, using the included Stackdriver driver,
and also assumes Gorilla Mux.

````
import (
  "github.com/gorilla/mux"
  "github.com/efixler/stats"
  "github.com/efixler/stats/stackdriver"
)

func init() {
  router := mux.NewRouter()
  router.Use(stats.Metrics(stackdriver.Sink))
  router.Use(stats.TimeRequests)
}

func SomeHandler(w http.ResponseWriter, r *http.Request) {
  stats.StartTimer(r.Context(), "some_handler_timer")
  // ... do work
  stats.FinishTimer(r.Context(), "some_handler_timer")
  stats.Increment(r.Context(), "some_handler_counter")
}
 ````

See the [Godoc](https://godoc.org/github.com/efixler/stats) for details and more examples. 
