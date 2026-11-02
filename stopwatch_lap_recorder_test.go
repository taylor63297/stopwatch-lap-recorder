package stopwatchlaprecorder

import (
	"strings"
	"testing"
	"time"
)

// fakeClock is a deterministic clock. Tests advance it explicitly.
type fakeClock struct {
	t time.Time
}

func (c *fakeClock) Now() time.Time { return c.t }
func (c *fakeClock) Advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func newFake() *fakeClock {
	return &fakeClock{t: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func TestStartLapStopElapsed(t *testing.T) {
	c := newFake()
	sw := New(c.Now)

	sw.Start()
	c.Advance(2 * time.Second)
	sw.Lap("warmup")
	c.Advance(3 * time.Second)
	sw.Lap("sprint")
	sw.Stop()

	laps := sw.Laps()
	if len(laps) != 2 {
		t.Fatalf("got %d laps, want 2", len(laps))
	}
	if laps[0].Name != "warmup" || laps[1].Name != "sprint" {
		t.Fatalf("unexpected lap names: %q, %q", laps[0].Name, laps[1].Name)
	}
	if laps[0].Duration != 2*time.Second {
		t.Errorf("warmup = %v, want 2s", laps[0].Duration)
	}
	if laps[1].Duration != 3*time.Second {
		t.Errorf("sprint = %v, want 3s", laps[1].Duration)
	}
	if sw.Elapsed() != 5*time.Second {
		t.Errorf("elapsed = %v, want 5s", sw.Elapsed())
	}
}

func TestStopFreezesElapsed(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(1 * time.Second)
	sw.Stop()
	c.Advance(10 * time.Second)
	if sw.Elapsed() != 1*time.Second {
		t.Errorf("elapsed after stop = %v, want 1s", sw.Elapsed())
	}
}

func TestDoubleStartAndDoubleStopAreNoOps(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(1 * time.Second)
	sw.Start() // ignored
	c.Advance(1 * time.Second)
	sw.Stop()
	sw.Stop() // ignored
	if sw.Elapsed() != 2*time.Second {
		t.Errorf("elapsed = %v, want 2s", sw.Elapsed())
	}
}

func TestLapBeforeStartIsIgnored(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Lap("too early")
	if len(sw.Laps()) != 0 {
		t.Fatalf("expected no laps before Start")
	}
}

func TestEmptyLapNameIsIgnored(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(1 * time.Second)
	sw.Lap("")
	if len(sw.Laps()) != 0 {
		t.Fatalf("expected empty name to be ignored")
	}
}

func TestLapAfterStopUsesStopTime(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(2 * time.Second)
	sw.Stop()
	c.Advance(5 * time.Second)
	sw.Lap("late")
	laps := sw.Laps()
	if len(laps) != 1 {
		t.Fatalf("got %d laps, want 1", len(laps))
	}
	if laps[0].Duration != 2*time.Second {
		t.Errorf("late lap = %v, want 2s", laps[0].Duration)
	}
}

func TestNegativeClockDeltaClampsToZero(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(-1 * time.Second) // clock went backwards
	if sw.Elapsed() != 0 {
		t.Errorf("elapsed = %v, want 0", sw.Elapsed())
	}
	sw.Lap("backwards")
	if sw.Laps()[0].Duration != 0 {
		t.Errorf("lap = %v, want 0", sw.Laps()[0].Duration)
	}
}

func TestSummaryContainsRowsAndTotal(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(1 * time.Second)
	sw.Lap("a")
	c.Advance(2 * time.Second)
	sw.Lap("b")
	sw.Stop()

	summary := sw.Summary()
	for _, want := range []string{"Name", "Lap", "Cumulative", "a", "b", "TOTAL"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary missing %q:\n%s", want, summary)
		}
	}
	if !strings.Contains(summary, "3s") {
		t.Errorf("summary missing total 3s:\n%s", summary)
	}
}

func TestSummaryBeforeStartIsEmptyTable(t *testing.T) {
	sw := New(newFake().Now)
	summary := sw.Summary()
	if !strings.Contains(summary, "TOTAL") {
		t.Errorf("summary should still have a TOTAL row:\n%s", summary)
	}
	if strings.Contains(summary, "a ") {
		t.Errorf("summary should not contain lap rows:\n%s", summary)
	}
}

func TestSortLapsByNameIsStable(t *testing.T) {
	laps := []Lap{
		{Name: "b", Duration: 1 * time.Second},
		{Name: "a", Duration: 2 * time.Second},
		{Name: "b", Duration: 3 * time.Second},
	}
	sorted := SortLapsByName(laps)
	if sorted[0].Name != "a" {
		t.Fatalf("first = %q, want a", sorted[0].Name)
	}
	if sorted[1].Duration != 1*time.Second || sorted[2].Duration != 3*time.Second {
		t.Errorf("stable order broken: %v", sorted)
	}
	if laps[0].Name != "b" {
		t.Errorf("SortLapsByName mutated its input")
	}
}

func TestSumLaps(t *testing.T) {
	laps := []Lap{{Duration: 1 * time.Second}, {Duration: 2 * time.Second}}
	if got := SumLaps(laps); got != 3*time.Second {
		t.Errorf("SumLaps = %v, want 3s", got)
	}
	if got := SumLaps(nil); got != 0 {
		t.Errorf("SumLaps(nil) = %v, want 0", got)
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(1500 * time.Millisecond); got != "1.500s" {
		t.Errorf("FormatDuration = %q, want 1.500s", got)
	}
}

func TestAlmostEqual(t *testing.T) {
	if !AlmostEqual(1*time.Second, 1001*time.Millisecond, 5*time.Millisecond) {
		t.Errorf("expected durations within tolerance to be equal")
	}
	if AlmostEqual(1*time.Second, 2*time.Second, 5*time.Millisecond) {
		t.Errorf("expected durations outside tolerance to differ")
	}
}

func TestMeanDurationEmpty(t *testing.T) {
	if got := meanDuration(nil); got != 0 {
		t.Errorf("meanDuration(nil) = %v, want 0", got)
	}
	if got := meanDuration([]Lap{{Duration: 2 * time.Second}, {Duration: 4 * time.Second}}); got != 3*time.Second {
		t.Errorf("meanDuration = %v, want 3s", got)
	}
}

func TestLapsReturnsCopy(t *testing.T) {
	c := newFake()
	sw := New(c.Now)
	sw.Start()
	c.Advance(1 * time.Second)
	sw.Lap("a")
	laps := sw.Laps()
	laps[0].Name = "mutated"
	if sw.Laps()[0].Name != "a" {
		t.Errorf("Laps returned a slice that aliases internal state")
	}
}
