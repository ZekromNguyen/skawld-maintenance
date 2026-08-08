package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	monitoringpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/adapter/postgres"
	monitoringdomain "github.com/ZekromNguyen/skawld-maintenance/internal/monitoring/domain"
	"github.com/go-chi/chi/v5"
)

func mountMonitoringRoutes(router chi.Router, store monitoringpostgres.Store) {
	router.Get("/monitoring/summary", monitoringSummary(store))
	router.Get("/monitoring/metrics", monitoringMetrics(store))
	router.Get("/monitoring/alerts", monitoringAlerts(store))
	router.Put("/monitoring/thresholds", monitoringThresholdsUpsert(store))
}

type monitoringSummaryEntry struct {
	MetricKey   string    `json:"metric_key"`
	Status      string    `json:"status"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Group       string    `json:"group"`
	SiteID      string    `json:"site_id,omitempty"`
	CollectedAt time.Time `json:"collected_at,omitempty"`
	OpenAlerts  int       `json:"open_alerts"`
}

func monitoringSummary(store monitoringpostgres.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		if !principal.Has(identitydomain.PermissionMonitoringRead) {
			writeProblem(w, http.StatusForbidden, "Forbidden",
				"monitoring read access is required")
			return
		}
		thresholds, err := store.ListThresholds(r.Context(), principal.OrganizationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Monitoring summary failed",
			})
			return
		}
		openAlerts, err := store.ListAlerts(r.Context(), principal.OrganizationID, true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Monitoring summary failed",
			})
			return
		}
		alertsByKey := map[string]int{}
		for _, alert := range openAlerts {
			alertsByKey[alert.MetricKey]++
		}
		entries := make([]monitoringSummaryEntry, 0, len(monitoringdomain.Catalog))
		for key, entry := range monitoringdomain.Catalog {
			value, collectedAt, ok, err := latestOrgWideMetric(r, store, principal.OrganizationID, key, entry.Granularity)
			if err != nil || !ok {
				continue // no data yet: the card is omitted until the job writes a row
			}
			warn, crit, comparator, enabled := effectiveThreshold(thresholds, key)
			status := monitoringdomain.StatusPass
			if enabled {
				status = monitoringdomain.Evaluate(value, warn, crit, comparator)
			}
			entries = append(entries, monitoringSummaryEntry{
				MetricKey: key, Status: string(status), Value: value,
				Unit: entry.Unit, Group: entry.Group,
				CollectedAt: collectedAt, OpenAlerts: alertsByKey[key],
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": entries})
	}
}

func latestOrgWideMetric(
	r *http.Request,
	store monitoringpostgres.Store,
	orgID, key string,
	granularity monitoringdomain.Granularity,
) (float64, time.Time, bool, error) {
	var value float64
	var collectedAt time.Time
	err := store.Pool.QueryRow(r.Context(), `
		SELECT value::float8, collected_at FROM monitoring_metrics
		WHERE organization_id = $1::uuid AND metric_key = $2
		  AND granularity = $3 AND site_id IS NULL
		ORDER BY collected_at DESC LIMIT 1
	`, orgID, key, granularity).Scan(&value, &collectedAt)
	if err != nil {
		return 0, time.Time{}, false, nil // no row yet; not an error
	}
	return value, collectedAt, true, nil
}

func effectiveThreshold(
	thresholds []monitoringpostgres.Threshold,
	key string,
) (warn, crit float64, comparator monitoringdomain.Comparator, enabled bool) {
	for _, row := range thresholds {
		if row.MetricKey != key || row.SiteID != "" {
			continue
		}
		return row.WarnValue, row.CritValue,
			monitoringdomain.Comparator(row.Comparator), row.Enabled
	}
	warn, crit, comparator, enabled, _ = monitoringdomain.DefaultThresholds(key)
	return warn, crit, comparator, enabled
}

func monitoringMetrics(store monitoringpostgres.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		if !principal.Has(identitydomain.PermissionMonitoringRead) {
			writeProblem(w, http.StatusForbidden, "Forbidden",
				"monitoring read access is required")
			return
		}
		key := strings.TrimSpace(r.URL.Query().Get("metric_key"))
		if key == "" || !monitoringdomain.InCatalog(key) {
			writeProblem(w, http.StatusBadRequest, "Bad Request",
				"metric_key must name a catalog metric")
			return
		}
		siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
		if siteID != "" && !validUUIDParam(w, siteID, "site ID") {
			return
		}
		from := time.Now().Add(-30 * 24 * time.Hour)
		to := time.Now().Add(time.Hour)
		if raw := r.URL.Query().Get("from"); raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				from = parsed
			}
		}
		if raw := r.URL.Query().Get("to"); raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				to = parsed
			}
		}
		rows, err := store.MetricsForRange(r.Context(), principal.OrganizationID, siteID, key, from, to)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Monitoring metrics failed",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": rows})
	}
}

func monitoringAlerts(store monitoringpostgres.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		if !principal.Has(identitydomain.PermissionMonitoringRead) {
			writeProblem(w, http.StatusForbidden, "Forbidden",
				"monitoring read access is required")
			return
		}
		state := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))
		openOnly := state != "RESOLVED" // default and "OPEN" show open events
		alerts, err := store.ListAlerts(r.Context(), principal.OrganizationID, openOnly)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Monitoring alerts failed",
			})
			return
		}
		siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
		if siteID != "" {
			filtered := alerts[:0]
			for _, alert := range alerts {
				if alert.SiteID == siteID || alert.SiteID == "" {
					filtered = append(filtered, alert)
				}
			}
			alerts = filtered
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": alerts})
	}
}

type thresholdRequest struct {
	MetricKey  string  `json:"metric_key"`
	SiteID     string  `json:"site_id,omitempty"`
	Comparator string  `json:"comparator"`
	WarnValue  float64 `json:"warn_value"`
	CritValue  float64 `json:"crit_value"`
	Enabled    *bool   `json:"enabled"`
}

func monitoringThresholdsUpsert(store monitoringpostgres.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		if !principal.Has(identitydomain.PermissionMonitoringConfigure) {
			writeProblem(w, http.StatusForbidden, "Forbidden",
				"monitoring configure access is required")
			return
		}
		var request thresholdRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid threshold body")
			return
		}
		if !monitoringdomain.InCatalog(request.MetricKey) {
			writeProblem(w, http.StatusBadRequest, "Bad Request", "unknown metric_key")
			return
		}
		comparator := monitoringdomain.Comparator(request.Comparator)
		switch comparator {
		case monitoringdomain.ComparatorGT, monitoringdomain.ComparatorGTE,
			monitoringdomain.ComparatorLT, monitoringdomain.ComparatorLTE:
		default:
			writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid comparator")
			return
		}
		if (comparator == monitoringdomain.ComparatorGT ||
			comparator == monitoringdomain.ComparatorGTE) &&
			request.CritValue < request.WarnValue {
			writeProblem(w, http.StatusBadRequest, "Bad Request", "crit_value must be >= warn_value")
			return
		}
		if (comparator == monitoringdomain.ComparatorLT ||
			comparator == monitoringdomain.ComparatorLTE) &&
			request.CritValue > request.WarnValue {
			writeProblem(w, http.StatusBadRequest, "Bad Request", "crit_value must be <= warn_value")
			return
		}
		enabled := true
		if request.Enabled != nil {
			enabled = *request.Enabled
		}
		if err := store.UpsertThreshold(r.Context(), monitoringpostgres.Threshold{
			ID: principal.ID, OrganizationID: principal.OrganizationID,
			SiteID: request.SiteID, MetricKey: request.MetricKey,
			Comparator: request.Comparator, WarnValue: request.WarnValue,
			CritValue: request.CritValue, Enabled: enabled,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Threshold upsert failed",
			})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
