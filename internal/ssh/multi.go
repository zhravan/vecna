package ssh

import (
	"context"
	"sync"
)

type CommandTarget struct {
	Host         Host
	Password     string
	SkipKey      bool
	JumpHost     *Host
	JumpPassword string
	JumpSkipKey  bool
}

type CommandResult struct {
	Host   Host
	Output string
	Err    error
	Index  int
}

// RunCommandParallel executes the same command across hosts concurrently.
// Results retain the input order so the UI can aggregate deterministically.
func RunCommandParallel(ctx context.Context, targets []CommandTarget, command string, concurrency int) []CommandResult {
	if concurrency < 1 {
		concurrency = 4
	}
	if concurrency > len(targets) && len(targets) > 0 {
		concurrency = len(targets)
	}
	results := make([]CommandResult, len(targets))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Add(1)
		go func(i int, target CommandTarget) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[i] = CommandResult{Host: target.Host, Index: i, Err: ctx.Err()}
				return
			}
			defer func() { <-sem }()

			outputCh := make(chan struct {
				output string
				err    error
			}, 1)
			go func() {
				out, err := RunCommand(target.Host, target.Password, target.SkipKey, target.JumpHost, target.JumpPassword, target.JumpSkipKey, command)
				outputCh <- struct {
					output string
					err    error
				}{out, err}
			}()
			select {
			case r := <-outputCh:
				results[i] = CommandResult{Host: target.Host, Output: r.output, Err: r.err, Index: i}
			case <-ctx.Done():
				results[i] = CommandResult{Host: target.Host, Index: i, Err: ctx.Err()}
			}
		}(i, target)
	}
	wg.Wait()
	return results
}
