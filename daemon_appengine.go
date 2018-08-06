// +build appengine

package stats

import (
	"context"
	"google.golang.org/appengine/runtime"
)

// See https://cloud.google.com/appengine/docs/standard/go/modules/runtime#RunInBackground
// for important information on scaling model constraints.
func startEventsListener(ctx context.Context, f func(context.Context)) error {
	return runtime.RunInBackground(ctx, f)
}
