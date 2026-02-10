package vre

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// BrowserPool manages a pool of Chrome browser contexts for reuse
type BrowserPool struct {
	allocCtx   context.Context
	browsers   chan *browserInstance
	maxSize    int
	mu         sync.Mutex
	closed     bool
	activeCount int
}

// browserInstance represents a single browser instance
type browserInstance struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBrowserPool creates a new browser pool
func NewBrowserPool(allocCtx context.Context, size int) (*BrowserPool, error) {
	if size <= 0 {
		size = 3
	}

	pool := &BrowserPool{
		allocCtx:   allocCtx,
		browsers:   make(chan *browserInstance, size),
		maxSize:    size,
		closed:     false,
	}

	// Warm up the pool with initial instances
	for i := 0; i < size; i++ {
		instance, err := pool.createInstance()
		if err != nil {
			// Clean up any created instances
			pool.Close()
			return nil, fmt.Errorf("failed to warm up pool: %w", err)
		}
		pool.browsers <- instance
	}

	return pool, nil
}

// createInstance creates a new browser instance
func (p *BrowserPool) createInstance() (*browserInstance, error) {
	ctx, cancel := chromedp.NewContext(p.allocCtx)

	// Initialize the browser with a simple navigation
	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	return &browserInstance{
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Acquire gets a browser context from the pool
func (p *BrowserPool) Acquire(ctx context.Context) (context.Context, func(), error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, nil, fmt.Errorf("pool is closed")
	}
	p.mu.Unlock()

	select {
	case instance := <-p.browsers:
		p.mu.Lock()
		p.activeCount++
		p.mu.Unlock()

		// Return the browser context with a release function
		release := func() {
			p.mu.Lock()
			p.activeCount--
			p.mu.Unlock()

			// Return to pool if not closed
			p.mu.Lock()
			closed := p.closed
			p.mu.Unlock()

			if !closed {
				select {
				case p.browsers <- instance:
				default:
					// Pool is full, close this instance
					instance.cancel()
				}
			} else {
				instance.cancel()
			}
		}

		return instance.ctx, release, nil

	case <-ctx.Done():
		return nil, nil, ctx.Err()

	case <-time.After(5 * time.Second):
		// Timeout waiting for a browser - try to create a new one
		p.mu.Lock()
		if p.activeCount < p.maxSize*2 { // Allow some overflow
			p.mu.Unlock()
			instance, err := p.createInstance()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create browser: %w", err)
			}

			p.mu.Lock()
			p.activeCount++
			p.mu.Unlock()

			release := func() {
				p.mu.Lock()
				p.activeCount--
				closed := p.closed
				p.mu.Unlock()

				if !closed {
					select {
					case p.browsers <- instance:
					default:
						instance.cancel()
					}
				} else {
					instance.cancel()
				}
			}

			return instance.ctx, release, nil
		}
		p.mu.Unlock()
		return nil, nil, fmt.Errorf("pool exhausted and timeout waiting for browser")
	}
}

// Size returns the current pool size
func (p *BrowserPool) Size() int {
	return len(p.browsers)
}

// ActiveCount returns the number of active (checked out) browsers
func (p *BrowserPool) ActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount
}

// Close closes all browsers in the pool
func (p *BrowserPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	// Drain and close all browsers
	close(p.browsers)
	for instance := range p.browsers {
		instance.cancel()
	}
}

// Stats returns pool statistics
type PoolStats struct {
	Available   int `json:"available"`
	Active      int `json:"active"`
	MaxSize     int `json:"max_size"`
}

// Stats returns current pool statistics
func (p *BrowserPool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return PoolStats{
		Available:   len(p.browsers),
		Active:      p.activeCount,
		MaxSize:     p.maxSize,
	}
}
