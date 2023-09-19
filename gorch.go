package gorch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/joinhuman/gorch/internal/launcher"
)

type Gorch struct {
	ctx       context.Context
	cancel    context.CancelFunc
	mu        *sync.Mutex
	launchers launchers
}

type launchers []struct {
	launcher   launcher.Launcher
	background bool
}

func New(ctx context.Context) *Gorch {
	ctx, cancel := context.WithCancel(ctx)

	return &Gorch{
		ctx:       ctx,
		cancel:    cancel,
		mu:        &sync.Mutex{},
		launchers: launchers{},
	}
}

// Register func registers an interface to initiate a service with smooth shutdown functionality.
// It attempts to call Start(ctx context.Context) and Stop(ctx context.Context) methods of the service to facilitate this.
// If any method is not present in the service interface, the launcher will not call it.
func (g *Gorch) Register(service any) {
	g.register(service, false)
}

// RegisterBackGround does the same as Register except for it is not acknowledging `started`
// and `stopped` states for the service.
func (g *Gorch) RegisterBackGround(service any) {
	g.register(service, true)
}

// Run func invokes all registered launchers and waits for error to be returned
// from any of those or ctx.Done().
func (g *Gorch) Run() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var errMsgs []string
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	wg.Add(len(g.launchers))

	for _, l := range g.launchers {
		go func(l launcher.Launcher) {
			defer wg.Done()

			if err := l.Launch(); err != nil {
				mu.Lock()
				errMsgs = append(errMsgs, err.Error())
				mu.Unlock()
			}

			l.WaitStop()
		}(l.launcher)

		if !l.background {
			l.launcher.WaitStart()
		}
	}

	wg.Wait()

	return fmt.Errorf("launchers: %v", errMsgs)
}

func (g *Gorch) register(service any, background bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	var (
		starter launcher.Starter
		stopper launcher.Stopper
	)

	if s, ok := service.(launcher.Starter); ok {
		starter = s
	}

	if s, ok := service.(launcher.Stopper); ok {
		stopper = s
	}

	g.launchers = append(g.launchers, struct {
		launcher   launcher.Launcher
		background bool
	}{
		launcher:   launcher.New(g.ctx, g.cancel, starter, stopper, time.Second*5),
		background: background,
	})
}
