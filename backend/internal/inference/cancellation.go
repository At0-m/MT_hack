package inference

import "context"

func runCancelable(ctx context.Context, run func() error, terminate func() error) error {
	terminated := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		_ = terminate()
		close(terminated)
	})
	// Join the callback before the caller destroys the native per-run options.
	defer func() {
		if !stop() {
			<-terminated
		}
	}()
	err := run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
