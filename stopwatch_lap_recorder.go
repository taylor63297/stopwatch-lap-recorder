// Package stopwatchlaprecorder implements a stopwatch that records named laps
// and produces a summary table.
//
// The package name matches the module name so that all source files can live in
// the root package, as required by the project layout.
package stopwatchlaprecorder

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Clock is the minimal surface of time.Time that the stopwatch needs. Taking a
// clock function as a constructor argument keeps the tests deterministic: they
// can advance a fake clock instead of sleeping on the wall clock.
type Clock func() time.Time

// Lap is a single named segment of elapsed time.
type Lap struct {
	Name     string
	Duration time.Duration
}

// Stopwatch accumulates elapsed time and named laps.
//
// The zero value is not usable; construct one with New.
type Stopwatch struct {
	now      Clock
	started  bool
	start    time.Time
	stopped  bool
	stopTime time.Time
	laps     []Lap
}

// New returns a Stopwatch that reads the current time from now.
//
// Passing the clock in explicitly is the only way to make lap durations
// reproducible in tests without sleeping.
func New(now Clock) *Stopwatch {
	if now == nil {
		now = time.Now
	}
	return &Stopwatch{now: now}
}

// Start begins timing. Starting an already-started stopwatch is a no-op.
func (s *Stopwatch) Start() {
	if s.started {
		return
	}
	s.started = true
	s.stopped = false
	s.start = s.now()
}

// Stop freezes the stopwatch. Stopping before Start or stopping twice is a
// no-op, which makes Stop safe to defer.
func (s *Stopwatch) Stop() {
	if !s.started || s.stopped {
		return
	}
	s.stopped = true
	s.stopTime = s.now()
}

// Lap records a named segment ending at the current time. The name must be
// non-empty; otherwise Lap is a no-op. Laps may be recorded while running or
// after Stop; after Stop the recorded duration is the time since the previous
// lap (or since Start).
func (s *Stopwatch) Lap(name string) {
	if !s.started || name == "" {
		return
	}
	end := s.now()
	if s.stopped {
		end = s.stopTime
	}
	var prev time.Time
	if len(s.laps) == 0 {
		prev = s.start
	} else {
		prev = s.lapEnd(len(s.laps) - 1)
	}
	d := end.Sub(prev)
	if d < 0 {
		d = 0
	}
	s.laps = append(s.laps, Lap{Name: name, Duration: d})
}

// lapEnd returns the absolute time at which lap i ended. Lap ends are stored
// implicitly as the cumulative sum of durations from start; this keeps the
// representation small and makes the summary independent of the clock.
func (s *Stopwatch) lapEnd(i int) time.Time {
	t := s.start
	for j := 0; j <= i; j++ {
		t = t.Add(s.laps[j].Duration)
	}
	return t
}

// Elapsed returns the total time measured so far. If the stopwatch is running,
// it uses the current clock reading; if stopped, it uses the frozen stop time.
func (s *Stopwatch) Elapsed() time.Duration {
	if !s.started {
		return 0
	}
	end := s.stopTime
	if !s.stopped {
		end = s.now()
	}
	d := end.Sub(s.start)
	if d < 0 {
		return 0
	}
	return d
}

// Laps returns a copy of the recorded laps in insertion order.
func (s *Stopwatch) Laps() []Lap {
	out := make([]Lap, len(s.laps))
	copy(out, s.laps)
	return out
}

// Summary renders the stopwatch state as a fixed-width table.
//
// The format is deliberately simple: a header, one row per lap, and a total
// row. The total is the stopwatch's Elapsed value, which may be larger than the
// sum of laps when time passed before the first lap or after the last one.
func (s *Stopwatch) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-20s %12s %12s\n", "Name", "Lap", "Cumulative")
	var cum time.Duration
	for _, l := range s.laps {
		cum += l.Duration
		fmt.Fprintf(&b, "%-20s %12s %12s\n", l.Name, l.Duration, cum)
	}
	fmt.Fprintf(&b, "%-20s %12s %12s\n", "TOTAL", "", s.Elapsed())
	return b.String()
}

// SortLapsByName returns a new slice of laps ordered by name. Ties keep the
// original insertion order thanks to sort.SliceStable. Sorting is a separate
// operation because the summary table is expected to show recording order.
func SortLapsByName(laps []Lap) []Lap {
	out := make([]Lap, len(laps))
	copy(out, laps)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// SumLaps returns the total of all lap durations.
func SumLaps(laps []Lap) time.Duration {
	var total time.Duration
	for _, l := range laps {
		total += l.Duration
	}
	return total
}

// FormatDuration renders a duration as seconds with three decimal places.
// Duration.String is precise but hard to compare across runs in a table, so the
// summary uses a fixed representation instead.
func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%.3fs", d.Seconds())
}

// AlmostEqual reports whether two durations differ by less than eps.
// Comparing durations directly with == is brittle when they are derived from
// floating point seconds; this helper keeps tests honest.
func AlmostEqual(a, b, eps time.Duration) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

// meanDuration returns the arithmetic mean of the given laps. It exists so the
// summary can be extended without changing callers; it returns 0 for an empty
// slice rather than dividing by zero.
func meanDuration(laps []Lap) time.Duration {
	if len(laps) == 0 {
		return 0
	}
	return SumLaps(laps) / time.Duration(len(laps))
}

// _ keeps math imported for callers that want to compare formatted output;
// the standard library does not offer a duration comparison with tolerance.
var _ = math.Abs
