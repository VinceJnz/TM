package httpProcessor

import (
	"context"
	"sync"
)

type requestLifecycle struct {
	mu     sync.Mutex
	closed bool
	next   uint64
	active map[uint64]context.CancelFunc
}

func newLifecycle() *requestLifecycle {
	return &requestLifecycle{active: map[uint64]context.CancelFunc{}}
}
func (l *requestLifecycle) begin(parent context.Context) (context.Context, func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, nil, context.Canceled
	}
	ctx, cancel := context.WithCancel(parent)
	l.next++
	id := l.next
	l.active[id] = cancel
	return ctx, func() { cancel(); l.mu.Lock(); delete(l.active, id); l.mu.Unlock() }, nil
}
func (l *requestLifecycle) stop() {
	l.mu.Lock()
	l.closed = true
	pending := make([]context.CancelFunc, 0, len(l.active))
	for _, cancel := range l.active {
		pending = append(pending, cancel)
	}
	l.mu.Unlock()
	for _, cancel := range pending {
		cancel()
	}
}
func (c *Client) requestContext() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}
func (c *Client) RequestContext(ctx context.Context, method, path string, rx, tx any) error {
	copy := *c
	copy.ctx = ctx
	return copy.Request(method, path, rx, tx)
}
func (c *Client) PageRequestContext(ctx context.Context, page *Page, method, path string, rx, tx any, callbacks ...func(error)) error {
	copy := *c
	copy.ctx = ctx
	return copy.PageRequest(page, method, path, rx, tx, callbacks...)
}
