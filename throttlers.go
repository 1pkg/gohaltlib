package gohalt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Throttler defines main interfaces
// for all derived throttlers, and
// defines main throttling lock/unlock flows.
type Throttler interface {
	Acquire(context.Context) error
	Release(context.Context) error
	Reset()
}

type tatomic struct {
	r uint64
	m uint64
}

func NewThrottlerAtomic(max uint64) *tatomic {
	return &tatomic{
		r: 0,
		m: max,
	}
}

func (t *tatomic) Acquire(context.Context) error {
	if atomic.CompareAndSwapUint64(&t.r, t.m, t.m) {
		return fmt.Errorf("throttler max running limit has been exceed %d", t.m)
	}
	atomic.AddUint64(&t.r, 1)
	return nil
}

func (t *tatomic) Release(context.Context) error {
	if atomic.CompareAndSwapUint64(&t.r, 0, 0) {
		return errors.New("throttler has nothing to release")
	}
	atomic.AddUint64(&t.r, ^uint64(0))
	return nil
}

func (t *tatomic) Reset() {
	atomic.StoreUint64(&t.r, 0)
}

type tblocking struct {
	r chan struct{}
	b sync.Mutex
}

func NewThrottlerBlocking(max uint64) *tblocking {
	return &tblocking{
		r: make(chan struct{}, max),
	}
}

func (t *tblocking) Acquire(context.Context) error {
	t.b.Lock()
	t.r <- struct{}{}
	t.b.Unlock()
	return nil
}

func (t *tblocking) Release(context.Context) error {
	select {
	case <-t.r:
	default:
		return errors.New("throttler has nothing to release")
	}
	return nil
}

func (t *tblocking) Reset() {
	t.b.Lock()
	for {
		select {
		case <-t.r:
		default:
			t.b.Unlock()
		}
	}
}

type ttimed struct {
	t Throttler
}

func NewThrottlerTimed(t Throttler, duration time.Duration) ttimed {
	go func() {
		tick := time.NewTicker(duration)
		defer tick.Stop()
		for {
			<-tick.C
			t.Reset()
		}
	}()
	return ttimed{t: t}
}

func (t ttimed) Acquire(ctx context.Context) error {
	return t.t.Acquire(ctx)
}

func (t ttimed) Release(ctx context.Context) error {
	return t.t.Release(ctx)
}

func (t ttimed) Reset() {
	t.t.Reset()
}
