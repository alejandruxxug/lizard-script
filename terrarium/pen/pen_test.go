package pen

import "testing"

func TestFlipY(t *testing.T) {
	// AppKit measures y from the bottom: 0 is the bottom edge.
	for _, c := range []struct{ y, h, want float64 }{{0, 600, 600}, {600, 600, 0}, {150, 600, 450}} {
		if got := FlipY(c.y, c.h); got != c.want {
			t.Errorf("FlipY(%v, %v) = %v, want %v", c.y, c.h, got, c.want)
		}
	}
}

func TestSnapshotFlips(t *testing.T) {
	s := snapshot(1.5, 10, 100, 400, PhaseDrag)
	if s.X != 10 || s.Y != 300 || s.T != 1.5 || s.Phase != PhaseDrag {
		t.Errorf("snapshot = %+v", s)
	}
	if s.Source == "" {
		t.Error("source should default to mouse")
	}
	if up := snapshot(2, 0, 0, 10, PhaseUp); up.Pressure != 0 {
		t.Errorf("up pressure = %v, want 0", up.Pressure)
	}
}

func TestQueueBoundAndDrain(t *testing.T) {
	Drain()
	for i := 0; i < maxQueue+100; i++ {
		record(Sample{T: float64(i)})
	}
	got := Drain()
	if len(got) != maxQueue {
		t.Fatalf("len = %d, want %d", len(got), maxQueue)
	}
	if got[0].T != 100 || got[len(got)-1].T != float64(maxQueue+99) {
		t.Errorf("kept %v..%v, want the newest %d in order", got[0].T, got[len(got)-1].T, maxQueue)
	}
	if again := Drain(); len(again) != 0 {
		t.Errorf("second drain returned %d samples", len(again))
	}
}
