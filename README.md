# stats
In-request counters and timers for golang web services

[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[godocs]: https://godoc.org/github.com/efixler/stats

More description here TBD.

## Installation

`go get github.com/efixler/stats`

Installing stats will also install the Stackdriver backend and command helpers. See the stackdriver/ subfolder 
for details.

## Usage
The usage sample below assumes Stackdriver as the storage backend for metrics data, using the included Stackdriver driver,
and also assumes Gorilla Mux.

````
import (
	"github.com/gorilla/mux"
  "github.com/efixler/taxat/stats"
	"github.com/efixler/taxat/stats/stackdriver"
)

func init() {
  router := mux.NewRouter()
  router.Use(stats.Metrics(stackdriver.Sink))
  router.Use(stats.TimeRequests)
}
 ````

See the [Godoc](https://godoc.org/github.com/efixler/config) for details and more examples. 
