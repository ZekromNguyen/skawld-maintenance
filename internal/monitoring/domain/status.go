package domain

type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusCrit Status = "CRIT"
)

type Comparator string

const (
	ComparatorGT  Comparator = "GT"
	ComparatorGTE Comparator = "GTE"
	ComparatorLT  Comparator = "LT"
	ComparatorLTE Comparator = "LTE"
)

func (c Comparator) Compare(value, threshold float64) bool {
	switch c {
	case ComparatorGT:
		return value > threshold
	case ComparatorGTE:
		return value >= threshold
	case ComparatorLT:
		return value < threshold
	case ComparatorLTE:
		return value <= threshold
	default:
		return false
	}
}

// Evaluate returns the alert status for a metric value against warn/crit
// thresholds. CRIT takes precedence over WARN; equality counts as passing for
// GT/LT comparators.
func Evaluate(value, warn, crit float64, comparator Comparator) Status {
	if comparator.Compare(value, crit) {
		return StatusCrit
	}
	if comparator.Compare(value, warn) {
		return StatusWarn
	}
	return StatusPass
}
