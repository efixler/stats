// +build appengine

package stats

import (
	"context"
	"google.golang.org/appengine/runtime"
)

func startEventsListener(ctx context.Context, f func(context.Context)) error {
	return runtime.RunInBackground(ctx, f)
}
