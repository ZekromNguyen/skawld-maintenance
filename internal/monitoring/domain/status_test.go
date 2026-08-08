package domain

import "testing"

func TestEvaluatePrecedence(t *testing.T) {
	// crit first, then warn, then pass
	if got := Evaluate(10, 5, 7, ComparatorGT); got != StatusCrit {
		t.Fatalf("expected CRIT, got %s", got)
	}
	if got := Evaluate(6, 5, 7, ComparatorGT); got != StatusWarn {
		t.Fatalf("expected WARN, got %s", got)
	}
	if got := Evaluate(1, 5, 7, ComparatorGT); got != StatusPass {
		t.Fatalf("expected PASS, got %s", got)
	}
}

func TestEvaluateLT(t *testing.T) {
	if got := Evaluate(0, 1, 1, ComparatorLT); got != StatusCrit {
		t.Fatalf("expected CRIT, got %s", got)
	}
	if got := Evaluate(1, 1, 1, ComparatorLT); got != StatusPass {
		t.Fatalf("expected PASS for equal value, got %s", got)
	}
}

func TestCompareGTEAndLTE(t *testing.T) {
	if !ComparatorGTE.Compare(5, 5) {
		t.Fatal("GTE should include equality")
	}
	if !ComparatorLTE.Compare(5, 5) {
		t.Fatal("LTE should include equality")
	}
	if ComparatorGT.Compare(5, 5) {
		t.Fatal("GT should exclude equality")
	}
}

func TestCatalogContainsPhaseOneKeys(t *testing.T) {
	for _, key := range []string{
		"sys.worker_queue_depth", "sys.worker_retry_count",
		"sys.worker_failed_count", "sys.ingestion_failed_count",
		"sys.backup_freshness_hours", "sys.health_ready",
		"ops.loto_blocked_count", "ops.loto_verified_count",
		"ops.incident_max_age_hours", "ops.incident_backlog_count",
		"ops.mttr_hours",
	} {
		if !InCatalog(key) {
			t.Errorf("catalog missing %q", key)
		}
	}
}

func TestDefaultThresholdsShape(t *testing.T) {
	for key := range Catalog {
		warn, crit, cmp, _, ok := DefaultThresholds(key)
		if !ok {
			t.Errorf("no default threshold for %q", key)
		}
		if cmp == "" {
			t.Errorf("no comparator for %q", key)
		}
		if cmp == ComparatorGT && crit < warn {
			t.Errorf("key %q crit %v < warn %v", key, crit, warn)
		}
		if cmp == ComparatorLT && crit > warn {
			t.Errorf("key %q crit %v > warn %v", key, crit, warn)
		}
	}
}

func TestDefaultThresholdsEnabledAlertsForSafetyKeys(t *testing.T) {
	// safety-relevant keys must alert out of the box
	for _, key := range []string{
		"sys.worker_failed_count", "sys.backup_freshness_hours",
		"sys.health_ready", "sys.worker_retry_count",
		"sys.ingestion_failed_count", "ops.incident_max_age_hours",
	} {
		_, _, _, enabled, ok := DefaultThresholds(key)
		if !ok {
			t.Fatalf("missing default for %q", key)
		}
		if !enabled {
			t.Errorf("expected %q default threshold to be enabled", key)
		}
	}
	// informational keys must not alert until configured
	for _, key := range []string{
		"ops.incident_backlog_count", "ops.mttr_hours",
		"ops.loto_blocked_count", "ops.loto_verified_count",
		"sys.worker_queue_depth",
	} {
		_, _, _, enabled, ok := DefaultThresholds(key)
		if !ok {
			t.Fatalf("missing default for %q", key)
		}
		if enabled {
			t.Errorf("expected %q default threshold to be disabled", key)
		}
	}
}
