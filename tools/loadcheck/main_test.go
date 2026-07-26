package main

import (
	"testing"
	"time"
)

func TestPercentileIsBoundedAndDeterministic(t *testing.T) {
	t.Parallel()
	values := []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
	}
	if got := percentile(values, 0.95); got != 3*time.Millisecond {
		t.Fatalf("p95 = %s", got)
	}
	if got := percentile(nil, 0.95); got != 0 {
		t.Fatalf("empty percentile = %s", got)
	}
	if got := milliseconds(1500 * time.Microsecond); got != 1.5 {
		t.Fatalf("milliseconds = %v", got)
	}
}
