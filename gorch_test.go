package gorch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testService struct {
	done bool
	ch   chan struct{}
}

func NewT() *testService {
	return &testService{
		ch: make(chan struct{}),
	}
}

func (ts *testService) Start() error {
	tc := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-tc.C:
			ts.done = true
		case <-ts.ch:
			return nil
		}
	}
}

func (ts *testService) Stop() error {
	ts.ch <- struct{}{}

	return nil
}

func TestGorch(t *testing.T) {
	tt := NewT()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	gorch := New(ctx)

	gorch.Register(tt)

	err := gorch.Run()

	require.Error(t, err)
}
