package state

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// NewContext combines the context interface with a graceful exit
func NewContext() Context {
	bg, cancel := context.WithCancel(context.Background())
	return &ctx{
		Context: bg,
		cancel:  cancel,
		sigChan: make(chan os.Signal, 1),
	}
}

type Context interface {
	context.Context
	// Defer registers a blocking function executed on context cancellation.
	Defer(fn func())
	// Exit cancels the context and waits for registered closers to exit.
	// The application force quits if a second sigterm or sigquit is received.
	Exit()
	// AwaitExit blocks until a shutdown signal is received.
	AwaitExit()
}

type ctx struct {
	context.Context
	mu      sync.Mutex
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	sigChan chan os.Signal // use a separate signal channel per context
}

// Defer spawns a goroutine to wait for all pending closers to finish.
func (ctx *ctx) Defer(fn func()) {
	ctx.mu.Lock()
	ctx.wg.Add(1)
	ctx.mu.Unlock()
	defer ctx.wg.Done()
	<-ctx.Done()
	fn()
}

// Exit triggers the ctx.Done chan, thereby releasing any goroutines waiting on chan
func (ctx *ctx) Exit() {
	ctx.cancel()
	// press Ctrl_C again to force quit
	var closer = make(chan struct{})
	go func() {
		defer close(closer)
		ctx.mu.Lock()
		ctx.wg.Wait()
		ctx.mu.Unlock()
	}()
	// Awaits on ctx sig chan or Defer chan.
	// If signal received it quits or,
	// if all closers finish, exits gracefully.
	force := make(chan os.Signal, 1)
	signal.Notify(force, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-force:
		fmt.Println("force quitting")
	case <-ctx.sigChan: // wait for context signal channel to exit
		fmt.Println("gracefully quitting")
	case <-closer:
	}
}

// AwaitExit blocks till an interrupt is received or context closed
func (ctx *ctx) AwaitExit() {
	exit, done := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer done()
	<-exit.Done()
	ctx.Exit()
}
