package main

import "context"

func Start[T Runner](ctx context.Context, r T) (T, <-chan error) {
	c := make(chan error, 0)
	go func() {
		if err := r.Run(ctx); err != nil {
			if err != nil {
				c <- err
			}
		}
		close(c)
	}()
	return r, c
}

type Runner interface {
	Run(context.Context) error
}
