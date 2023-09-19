package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testStarter struct {
	done bool
}

func (ts *testStarter) Start(ctx context.Context) error {
	ts.done = true

	return nil
}

type testStopper struct {
	done bool
}

func (ts *testStopper) Stop(ctx context.Context) error {
	ts.done = true

	return nil
}

func TestLauncherCanceled(t *testing.T) {
	var (
		tStarter testStarter
		tStopper testStopper
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()

	l := New(ctx, cancel, &tStarter, &tStopper, time.Second)

	err := l.Launch()
	require.ErrorIs(t, err, context.Canceled)

	l.WaitStart()
	require.True(t, tStarter.done)
}

func TestLauncherDeadlineExceeded(t *testing.T) {
	var (
		tStarter testStarter
		tStopper testStopper
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	l := New(ctx, cancel, &tStarter, &tStopper, time.Second*3)

	err := l.Launch()
	require.ErrorIs(t, err, context.DeadlineExceeded)

	l.WaitStart()
	require.True(t, tStarter.done)

	l.WaitStop()
	require.True(t, tStopper.done)
}
