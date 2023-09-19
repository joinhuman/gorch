package launcher

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type (
	Starter interface {
		Start(ctx context.Context) error
	}
	Stopper interface {
		Stop(ctx context.Context) error
	}
)

// Launcher interface determines the way to gracefully merge Starter and Stopper functionality.
type Launcher interface {
	Launch() error
	WaitStart()
	WaitStop()
}

type launcher struct {
	ctx             context.Context
	cancel          context.CancelFunc
	starter         Starter
	started         atomic.Bool
	stopper         Stopper
	stopped         atomic.Bool
	gracefulTimeOut time.Duration
}

func New(ctx context.Context, cancel context.CancelFunc, starter Starter, stopper Stopper, gracefulTimeOut time.Duration) *launcher {
	return &launcher{
		ctx:             ctx,
		cancel:          cancel,
		starter:         starter,
		started:         atomic.Bool{},
		stopper:         stopper,
		stopped:         atomic.Bool{},
		gracefulTimeOut: gracefulTimeOut,
	}
}

func (l *launcher) Launch() error {
	var err error

	if l.starter != nil {
		go func() {
			if startErr := l.starter.Start(l.ctx); startErr != nil {
				err = fmt.Errorf("starter.Start: %w", startErr)
				l.cancel()
			}

			l.started.Store(true)
		}()
	} else {
		l.started.Store(true)
	}

	if l.stopper != nil {
		errCh := make(chan error)

		<-l.ctx.Done()

		go func() {
			if stopErr := l.stopper.Stop(l.ctx); stopErr != nil {
				errCh <- fmt.Errorf("stopper.Stop: %w", stopErr)
				return
			}

			errCh <- nil
		}()

		select {
		case stopErr := <-errCh:
			if stopErr == nil {
				stopErr = l.ctx.Err()
			}

			err = errors.Join(err, fmt.Errorf("stopper.Stop: %w", stopErr))
		case <-time.After(l.gracefulTimeOut):
		}

		l.stopped.Store(true)
	} else {
		l.stopped.Store(true)
	}

	return err
}

func (l *launcher) WaitStart() {
	for !l.started.Load() {
		time.Sleep(time.Second / 2)
	}
}

func (l *launcher) WaitStop() {
	for !l.stopped.Load() {
		time.Sleep(time.Second / 2)
	}
}
