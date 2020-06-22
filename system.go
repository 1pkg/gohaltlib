package gohalt

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

type Stats interface {
	Stats() (alloc uint64, system uint64, avgpause uint64, avgusage float64)
}

type cachedstats struct {
	alloc    uint64
	system   uint64
	avgpause uint64
	avgusage float64
}

func NewCachedStats(ctx context.Context, duration time.Duration) (*cachedstats, error) {
	s := &cachedstats{}
	loop(ctx, duration, s.sync)
	return s, s.sync(ctx)
}

func (s cachedstats) Stats() (alloc uint64, system uint64, avgpause uint64, avgusage float64) {
	return s.alloc, s.system, s.avgpause, s.avgusage
}

func (s *cachedstats) sync(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	s.alloc = stats.Alloc
	s.system = stats.Sys
	for _, p := range stats.PauseNs {
		s.avgpause += p
	}
	s.avgpause /= 256
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if percents, err := cpu.Percent(10*time.Millisecond, true); err != nil {
		for _, p := range percents {
			s.avgusage += p
		}
		s.avgusage /= float64(len(percents))
	}
	return nil
}