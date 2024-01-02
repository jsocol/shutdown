package shutdown

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"
)

var timeout = 10 * time.Second

// In the event of an unclean shutdown, one of the following statuses will be
// returned:
const (
	StatusTimeoutExceeded   = 4 // the graceful timeout was exceeded
	StatusInterruptReceived = 3 // a second interrupt was received
	StatusTaskError         = 2 // one of the shutdown tasks returned an error
)

// SetTimeout sets the graceful shutdown timeout. It can be called at any point
// before the first interrupt signal is captured. The default is 10s.
func SetTimeout(d time.Duration) {
	timeout = d
}

// ShutdownTasks are called when an interrupt is received. They must
// successfully complete before the timeout is reached, or they will be
// abandoned. They receive a context.Context they can pass to any calls they
// may need to make. For example, [net/http.Server.Shutdown] implements this
// interface.
type ShutdownTask func(context.Context) error

// Listen takes any number of [ShutdownTask] functions and waits for an
// interrupt signal. When a signal is received, the tasks are executed
// concurrently. The tasks may be abandoned in the following cases:
//
// - The graceful shutdown timeout is reached (default: 10s). See [SetTimeout].
// - Another interrupt signal is received.
// - One of the tasks returns an error.
func Listen(tasks ...ShutdownTask) {
	sigchan := make(chan os.Signal, 1)
	errchan := make(chan error, 1)
	donechan := make(chan struct{})
	wg := sync.WaitGroup{}

	go func() {
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan

		slog.Info("shutting down", "timeout", timeout)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		for _, t := range tasks {
			wg.Add(1)
			go func(t ShutdownTask) {
				if err := t(ctx); err != nil {
					errchan <- err
				}
				wg.Done()
			}(t)
		}

		go func() {
			wg.Wait()
			close(donechan)
		}()

		select {
		case <-ctx.Done():
			// timeout
			slog.Error("shutdown timeout exceeded")
			os.Exit(StatusTimeoutExceeded)
		case <-sigchan:
			// a second interrupt
			slog.Warn("interrupt received; shutting down immediately")
			os.Exit(StatusInterruptReceived)
		case err := <-errchan:
			// task error occurred
			slog.Error("error during graceful shutdown", "error", err)
			os.Exit(StatusTaskError)
		case <-donechan:
			// success!
		}
	}()
}
