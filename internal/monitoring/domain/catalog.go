package domain

type Granularity string

const (
	GranularityLatest Granularity = "latest"
	GranularityDay    Granularity = "day"
)

// CatalogEntry describes one monitor metric. Group separates the UI tabs
// ("system" | "ops"); Direction tells the UI which way is good.
type CatalogEntry struct {
	Key         string
	Group       string
	DisplayName string
	Unit        string
	Direction   string // "down_is_good" | "up_is_good"
	Granularity Granularity
}

// Catalog is the fixed, reviewed set of Phase 1 monitor keys. New monitors
// are added here, in the collectors, and in the web catalog, never ad hoc.
var Catalog = map[string]CatalogEntry{
	"sys.worker_queue_depth": {
		Key: "sys.worker_queue_depth", Group: "system",
		DisplayName: "Worker queue depth", Unit: "jobs",
		Direction: "down_is_good", Granularity: GranularityLatest,
	},
	"sys.worker_retry_count": {
		Key: "sys.worker_retry_count", Group: "system",
		DisplayName: "Worker retries", Unit: "jobs",
		Direction: "down_is_good", Granularity: GranularityLatest,
	},
	"sys.worker_failed_count": {
		Key: "sys.worker_failed_count", Group: "system",
		DisplayName: "Worker failed jobs", Unit: "jobs",
		Direction: "down_is_good", Granularity: GranularityLatest,
	},
	"sys.ingestion_failed_count": {
		Key: "sys.ingestion_failed_count", Group: "system",
		DisplayName: "Ingestion failures", Unit: "docs",
		Direction: "down_is_good", Granularity: GranularityLatest,
	},
	"sys.backup_freshness_hours": {
		Key: "sys.backup_freshness_hours", Group: "system",
		DisplayName: "Backup freshness", Unit: "hours",
		Direction: "down_is_good", Granularity: GranularityLatest,
	},
	"sys.health_ready": {
		Key: "sys.health_ready", Group: "system",
		DisplayName: "Backend ready", Unit: "0/1",
		Direction: "up_is_good", Granularity: GranularityLatest,
	},
	"ops.loto_blocked_count": {
		Key: "ops.loto_blocked_count", Group: "ops",
		DisplayName: "LOTO blocks", Unit: "events",
		Direction: "down_is_good", Granularity: GranularityDay,
	},
	"ops.loto_verified_count": {
		Key: "ops.loto_verified_count", Group: "ops",
		DisplayName: "LOTO verifications", Unit: "events",
		Direction: "up_is_good", Granularity: GranularityDay,
	},
	"ops.incident_max_age_hours": {
		Key: "ops.incident_max_age_hours", Group: "ops",
		DisplayName: "Oldest open incident", Unit: "hours",
		Direction: "down_is_good", Granularity: GranularityDay,
	},
	"ops.incident_backlog_count": {
		Key: "ops.incident_backlog_count", Group: "ops",
		DisplayName: "Aged incident backlog", Unit: "incidents",
		Direction: "down_is_good", Granularity: GranularityDay,
	},
	"ops.mttr_hours": {
		Key: "ops.mttr_hours", Group: "ops",
		DisplayName: "MTTR", Unit: "hours",
		Direction: "down_is_good", Granularity: GranularityDay,
	},
}

func InCatalog(key string) bool {
	_, ok := Catalog[key]
	return ok
}

// DefaultThresholds returns the seeded threshold for a metric key.
// enabled=false keeps the metric informational (no alerts) until an operator
// configures a row in monitor_thresholds; the row's enabled=true overrides.
func DefaultThresholds(key string) (warn, crit float64, comparator Comparator, enabled, ok bool) {
	switch key {
	case "sys.worker_failed_count", "sys.ingestion_failed_count":
		return 0, 0, ComparatorGT, true, true // any failure above 0 is CRIT
	case "sys.worker_retry_count":
		return 5, 20, ComparatorGT, true, true
	case "sys.backup_freshness_hours":
		return 24, 48, ComparatorGT, true, true
	case "sys.health_ready":
		return 1, 1, ComparatorLT, true, true // value 0 (not ready) is CRIT
	case "ops.incident_max_age_hours":
		return 24, 72, ComparatorGT, true, true
	case "ops.incident_backlog_count", "ops.mttr_hours",
		"ops.loto_blocked_count", "ops.loto_verified_count",
		"sys.worker_queue_depth":
		return 0, 0, ComparatorGT, false, true // informational: no alerts until configured
	default:
		return 0, 0, "", false, false
	}
}
