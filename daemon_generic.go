// +build !appengine

package stats

import (
	"context"
)

func startEventsListener(ctx context.Context, f func(context.Context)) error {
	// in normal Go (e.g. non-appengine) environemnts we ignore the passed
	// context and use the background context instead,
	ctx = context.Background()
	go f(ctx)
	return nil
}
