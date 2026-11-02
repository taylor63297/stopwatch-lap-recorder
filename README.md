# stopwatch-lap-recorder

A stopwatch that records named laps and renders a cumulative summary table.

## Usage

```go
package main

import (
	"fmt"
	"time"

	sw "stopwatch-lap-recorder"
)

func main() {
	s := sw.New(time.Now)
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Lap("setup")
	time.Sleep(20 * time.Millisecond)
	s.Lap("work")
	s.Stop()

	fmt.Print(s.Summary())
}
```

## Why this exists

Timing code usually ends up as a pile of `time.Now()` calls and ad-hoc
subtractions scattered through a function. That is fine until you want to report
where the time went, at which point you are reconstructing laps from timestamps
by hand. This package keeps the running total and the named segments in one
place, and makes the result printable without a second pass.

The trade-off is that laps are stored as durations, not as absolute end times.
That keeps the summary independent of the clock and makes tests trivial, but it
means the package cannot tell you the wall-clock instant at which a lap ended.
If you need that, record it yourself alongside the lap name.

## Edge case you will hit

Calling `Lap` before `Start` is silently ignored, and so is an empty lap name.
This is deliberate: instrumentation code often runs in paths that may or may not
have started the stopwatch, and raising an error there would be more annoying
than useful. `Stop` is likewise idempotent. If you need to know whether a lap
was actually recorded, check the length of `Laps()`.

## API

- `New(now Clock) *Stopwatch` — construct a stopwatch. Pass `nil` to use
  `time.Now`; pass a fake clock in tests.
- `(*Stopwatch) Start()` — begin timing. Idempotent.
- `(*Stopwatch) Stop()` — freeze elapsed time. Idempotent.
- `(*Stopwatch) Lap(name string)` — record a named segment.
- `(*Stopwatch) Elapsed() time.Duration` — total measured time.
- `(*Stopwatch) Laps() []Lap` — a copy of the recorded laps, in order.
- `(*Stopwatch) Summary() string` — a fixed-width table with a total row.
- `SortLapsByName(laps []Lap) []Lap` — a stable copy sorted by name.
- `SumLaps(laps []Lap) time.Duration` — total of all lap durations.
- `FormatDuration(d time.Duration) string` — seconds with three decimals.
- `AlmostEqual(a, b, eps time.Duration) bool` — tolerance comparison.
- `Lap{Name string; Duration time.Duration}` — one recorded segment.
- `Clock func() time.Time` — the clock function type.
