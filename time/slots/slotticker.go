// Package slots includes ticker and timer-related functions for Ethereum consensus.
package slots

import (
	"time"

	"github.com/theQRL/qrysm/v4/config/params"
	"github.com/theQRL/qrysm/v4/consensus-types/primitives"
	prysmTime "github.com/theQRL/qrysm/v4/time"
)

// The Ticker interface defines a type which can expose a
// receive-only channel firing slot events.
type Ticker interface {
	C() <-chan primitives.Slot
	Done()
}

// SlotTicker is a special ticker for the beacon chain block.
// The channel emits over the slot interval, and ensures that
// the ticks are in line with the genesis time. This means that
// the duration between the ticks and the genesis time are always a
// multiple of the slot duration.
// In addition, the channel returns the new slot number.
type SlotTicker struct {
	c    chan primitives.Slot
	done chan struct{}
}

// C returns the ticker channel. Call Cancel afterwards to ensure
// that the goroutine exits cleanly.
func (s *SlotTicker) C() <-chan primitives.Slot {
	return s.c
}

// Done should be called to clean up the ticker.
func (s *SlotTicker) Done() {
	go func() {
		s.done <- struct{}{}
	}()
}

// NewSlotTicker starts and returns a new SlotTicker instance.
func NewSlotTicker(genesisTime time.Time, secondsPerSlot uint64) *SlotTicker {
	if genesisTime.IsZero() {
		panic("zero genesis time")
	}
	ticker := &SlotTicker{
		c:    make(chan primitives.Slot),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime, secondsPerSlot, prysmTime.Since, prysmTime.Until, time.After)
	return ticker
}

// NewSlotTickerWithOffset starts and returns a SlotTicker instance that allows a offset of time from genesis,
// entering a offset greater than secondsPerSlot is not allowed.
func NewSlotTickerWithOffset(genesisTime time.Time, offset time.Duration, secondsPerSlot uint64) *SlotTicker {
	if genesisTime.Unix() == 0 {
		panic("zero genesis time")
	}
	if offset > time.Duration(secondsPerSlot)*time.Second {
		panic("invalid ticker offset")
	}
	ticker := &SlotTicker{
		c:    make(chan primitives.Slot),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime.Add(offset), secondsPerSlot, prysmTime.Since, prysmTime.Until, time.After)
	return ticker
}

func (s *SlotTicker) start(
	genesisTime time.Time,
	secondsPerSlot uint64,
	since, until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time) {
	d := time.Duration(secondsPerSlot) * time.Second

	go func() {
		sinceGenesis := since(genesisTime)

		var nextTickTime time.Time
		var slot primitives.Slot
		if sinceGenesis < d {
			// Handle when the current time is before the genesis time.
			nextTickTime = genesisTime
			slot = 0
		} else {
			nextTick := sinceGenesis.Truncate(d) + d
			nextTickTime = genesisTime.Add(nextTick)
			slot = primitives.Slot(nextTick / d)
		}

		for {
			waitTime := until(nextTickTime)
			select {
			case <-after(waitTime):
				s.c <- slot
				slot++
				nextTickTime = nextTickTime.Add(d)
			case <-s.done:
				return
			}
		}
	}()
}

// startWithIntervals starts a ticker that emits a tick every slot at the
// prescribed intervals. The caller is responsible to make these intervals increasing and
// less than secondsPerSlot
func (s *SlotTicker) startWithIntervals(
	genesisTime time.Time,
	until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time,
	intervals []time.Duration) {
	go func() {
		slot := Since(genesisTime)
		slot++
		interval := 0
		nextTickTime := startFromTime(genesisTime, slot).Add(intervals[0])

		for {
			waitTime := until(nextTickTime)
			select {
			case <-after(waitTime):
				s.c <- slot
				interval++
				if interval == len(intervals) {
					interval = 0
					slot++
				}
				nextTickTime = startFromTime(genesisTime, slot).Add(intervals[interval])
			case <-s.done:
				return
			}
		}
	}()
}

// NewSlotTickerWithIntervals starts and returns a SlotTicker instance that allows
// several offsets of time from genesis,
// Caller is responsible to input the intervals in increasing order and none bigger or equal than
// SecondsPerSlot
func NewSlotTickerWithIntervals(genesisTime time.Time, intervals []time.Duration) *SlotTicker {
	if genesisTime.Unix() == 0 {
		panic("zero genesis time")
	}
	if len(intervals) == 0 {
		panic("at least one interval has to be entered")
	}
	slotDuration := time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	lastOffset := time.Duration(0)
	for _, offset := range intervals {
		if offset < lastOffset {
			panic("invalid decreasing offsets")
		}
		if offset >= slotDuration {
			panic("invalid ticker offset")
		}
		lastOffset = offset
	}
	ticker := &SlotTicker{
		c:    make(chan primitives.Slot),
		done: make(chan struct{}),
	}
	ticker.startWithIntervals(genesisTime, prysmTime.Until, time.After, intervals)
	return ticker
}
