// runtime flow methods
package rtflow

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

type (
	DeferredFunc func()
)

var (
	deferredFunctions           = make(chan DeferredFunc, 16)
	ErrTooManyDeferredFunctions = errors.New("too many deferred functions, increase channel capacity")
)

// Main starts function runFunc with specified context. The context will be canceled
// by SIGTERM or SIGINT signal (Ctrl+C for example)
// beforeExit function must be executed immediately before exit
func Main(runFunc func(ctx context.Context, cancel context.CancelFunc) error) {
	// context should be canceled while Int signal will be caught
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// main processing loop
	retChan := make(chan error, 1)
	go func() {
		err2 := runFunc(ctx, cancel)
		if err2 != nil {
			retChan <- err2
		}
		close(retChan)
	}()

	// Waiting signals from OS
	go func() {
		defer cancel()
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
		select {
		case sig := <-quit:
			log.Warn().Msgf("Signal '%s' was caught. Exiting", sig)
			return
		case <-ctx.Done():
			return
		}
	}()

	// Listening for the main loop response
	for e := range retChan {
		log.Info().Err(e).Msg("Exiting.")
	}

	close(deferredFunctions)
	for f := range deferredFunctions {
		f()
	}
}

// AddDefer adds global deferred function they must be executed before completely exiting the application
func AddDefer(deferredFunc DeferredFunc) {
	select {
	case deferredFunctions <- deferredFunc:
	default:
		panic(ErrTooManyDeferredFunctions) // it's good idea to increase channel capacity!
	}
}
