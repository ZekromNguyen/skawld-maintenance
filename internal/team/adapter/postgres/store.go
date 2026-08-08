package postgres

import (
	"context"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	teamapp "github.com/ZekromNguyen/skawld-maintenance/internal/team/application"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func (s Store) ListTeams(ctx context.Context, principal identitydomain.Principal) ([]teamapp.Team, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, name
		FROM teams
		WHERE organization_id = $1::uuid
		ORDER BY name
	`, principal.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	teams := []teamapp.Team{}
	for rows.Next() {
		var team teamapp.Team
		if err := rows.Scan(&team.ID, &team.Name); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, rows.Err()
}

func (s Store) ListPeople(ctx context.Context, principal identitydomain.Principal) ([]teamapp.Person, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT p.id::text, p.display_name
		FROM principals p
		JOIN memberships m ON m.principal_id = p.id
		WHERE m.organization_id = $1::uuid
		  AND (COALESCE(cardinality($2::uuid[]), 0) = 0 OR m.site_id = ANY($2::uuid[]) OR m.site_id IS NULL)
		ORDER BY p.display_name
	`, principal.OrganizationID, principal.SiteIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	people := []teamapp.Person{}
	for rows.Next() {
		var person teamapp.Person
		if err := rows.Scan(&person.ID, &person.DisplayName); err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, rows.Err()
}
