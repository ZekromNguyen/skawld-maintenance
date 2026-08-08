package httpserver

import (
	"net/http"
	"strings"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mountSummaryRoutes(router chi.Router, pool *pgxpool.Pool) {
	router.Get("/summary", summary(pool))
}

func summary(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromRequest(w, r)
		if !ok {
			return
		}
		if !principal.Has(identitydomain.PermissionIncidentRead) {
			writeProblem(w, http.StatusForbidden, "Forbidden", "summary requires incident read access")
			return
		}
		siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
		if siteID != "" && !validUUIDParam(w, siteID, "site ID") {
			return
		}

		row := pool.QueryRow(r.Context(), `
			SELECT
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
				    AND i.state = 'OPEN'),
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
				    AND i.state = 'IN_PROGRESS'),
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
				    AND i.state = 'RESOLVED'),
				(SELECT count(*) FROM incidents i
				  WHERE i.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
				    AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)),
				(SELECT count(*) FROM maintenance_executions e
				  WHERE e.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR e.site_id = ANY($2::uuid[]))
				    AND (nullif($3, '') IS NULL OR e.site_id = $3::uuid)
				    AND e.state = 'IN_PROGRESS'),
				(SELECT count(*) FROM asset_criticalities ac
				  WHERE ac.organization_id = $1::uuid
				    AND ac.superseded_at IS NULL
				    AND ac.rating = 'A'
				    AND EXISTS (
				      SELECT 1 FROM assets a
				      WHERE a.id = ac.asset_id
				        AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR a.site_id = ANY($2::uuid[]))
				        AND (nullif($3, '') IS NULL OR a.site_id = $3::uuid))),
				(SELECT count(*) FROM shift_handovers h
				  WHERE h.organization_id = $1::uuid
				    AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR h.site_id = ANY($2::uuid[]))
				    AND (nullif($3, '') IS NULL OR h.site_id = $3::uuid)
				    AND h.state IN ('DRAFT', 'SUBMITTED'))`,
			principal.OrganizationID, principal.SiteIDs, siteID,
		)

		var open, inProgress, resolved, total, activeExecutions, criticalAssets, pendingHandovers int
		if err := row.Scan(&open, &inProgress, &resolved, &total,
			&activeExecutions, &criticalAssets, &pendingHandovers); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Summary query failed",
			})
			return
		}

		severityCounts := map[string]int{}
		rows, err := pool.Query(r.Context(), `
			SELECT i.severity, count(*)
			FROM incidents i
			WHERE i.organization_id = $1::uuid
			  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR i.site_id = ANY($2::uuid[]))
			  AND (nullif($3, '') IS NULL OR i.site_id = $3::uuid)
			GROUP BY i.severity`,
			principal.OrganizationID, principal.SiteIDs, siteID,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": http.StatusInternalServerError, "title": "Summary query failed",
			})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var severity string
			var count int
			if err := rows.Scan(&severity, &count); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"status": http.StatusInternalServerError, "title": "Summary query failed",
				})
				return
			}
			severityCounts[severity] = count
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"open_incidents":        open,
			"in_progress_incidents": inProgress,
			"resolved_incidents":    resolved,
			"total_incidents":       total,
			"by_severity": map[string]int{
				"LOW":      severityCounts["LOW"],
				"MEDIUM":   severityCounts["MEDIUM"],
				"HIGH":     severityCounts["HIGH"],
				"CRITICAL": severityCounts["CRITICAL"],
			},
			"active_executions": activeExecutions,
			"critical_assets":   criticalAssets,
			"pending_handovers": pendingHandovers,
		})
	}
}
