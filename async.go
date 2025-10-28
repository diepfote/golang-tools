package main

import (
	"context"
	"fmt"
	"io"
)

func asyncReadPipe(pipe io.ReadCloser, pipeName string, errCh chan error, pipeBytesCh chan []byte, ctx context.Context) {
	go func(ctx context.Context) {
		b, err := io.ReadAll(pipe)
		if err != nil {
			if ctx != nil && ctx.Err() != context.DeadlineExceeded {
				errCh <- fmt.Errorf("Failed to read %s: %w", pipeName, err)
			} else {
				/* hacky way to prevent printing errors
				   if we run into a timeout
				   part 1 - stdout
				*/
				errCh <- fmt.Errorf("")
			}
		} else {
			pipeBytesCh <- b
		}
	}(ctx)
}
