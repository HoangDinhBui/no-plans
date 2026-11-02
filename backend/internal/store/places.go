package store

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/plans"
)

type Place struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Activity        string     `json:"activity"`
	Kind            string     `json:"kind"`
	Area            string     `json:"area"`
	Address         string     `json:"address"`
	Description     string     `json:"description"`
	BudgetMin       *int       `json:"budget_min"`
	BudgetMax       *int       `json:"budget_max"`
	PriceNote       string     `json:"price_note"`
	PeopleMin       int        `json:"people_min"`
	PeopleMax       int        `json:"people_max"`
	SourceURL       string     `json:"source_url"`
	SourceCheckedAt *time.Time `json:"source_checked_at,omitempty"`
	Active          bool       `json:"active"`
}
type PlaceFilter struct {
	Activity, Kind, Area, Cursor string
	MaxBudget                    *int
	People, Limit                int
}

func validActivity(value string) bool {
	switch value {
	case "food", "coffee", "drinks", "cinema", "sports", "other":
		return true
	}
	return false
}

// UpsertPlaces is an operator/seed interface, never exposed to ordinary HTTP users.
func (s *Store) UpsertPlaces(ctx context.Context, places []Place) error {
	if len(places) == 0 || len(places) > 500 {
		return plans.ErrInvalid
	}
	normalized := make([]Place, len(places))
	seen := map[string]bool{}
	for i, p := range places {
		p.ID = strings.ToLower(p.ID)
		p.Name = strings.TrimSpace(p.Name)
		p.Area = strings.TrimSpace(p.Area)
		p.Address = strings.TrimSpace(p.Address)
		if !ids.Valid(p.ID) || seen[p.ID] || p.Name == "" || utf8.RuneCountInString(p.Name) > 120 || p.Area == "" || utf8.RuneCountInString(p.Area) > 80 || !validActivity(p.Activity) || (p.Kind != "place" && p.Kind != "idea") || utf8.RuneCountInString(p.Address) > 500 || utf8.RuneCountInString(p.Description) > 1000 || utf8.RuneCountInString(p.PriceNote) > 200 || p.PeopleMin < 1 || p.PeopleMin > 1000 || p.PeopleMax < 0 || p.PeopleMax > 1000 || (p.PeopleMax != 0 && p.PeopleMax < p.PeopleMin) {
			return plans.ErrInvalid
		}
		seen[p.ID] = true
		if (p.BudgetMin == nil) != (p.BudgetMax == nil) {
			return plans.ErrInvalid
		}
		if p.BudgetMin != nil && (*p.BudgetMin < 0 || *p.BudgetMax < *p.BudgetMin || *p.BudgetMax > 1<<31-1 || strings.TrimSpace(p.PriceNote) == "") {
			return plans.ErrInvalid
		}
		if p.Kind == "place" && (p.SourceURL == "" || p.SourceCheckedAt == nil) {
			return plans.ErrInvalid
		}
		if p.SourceURL != "" {
			u, err := url.Parse(p.SourceURL)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || len(p.SourceURL) > 2048 {
				return plans.ErrInvalid
			}
		}
		normalized[i] = p
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, p := range normalized {
		_, err = tx.Exec(ctx, `INSERT INTO places(id,name,activity,kind,area,address,description,budget_min,budget_max,price_note,people_min,people_max,source_url,source_checked_at,active) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT(id) DO UPDATE SET name=excluded.name,activity=excluded.activity,kind=excluded.kind,area=excluded.area,address=excluded.address,description=excluded.description,budget_min=excluded.budget_min,budget_max=excluded.budget_max,price_note=excluded.price_note,people_min=excluded.people_min,people_max=excluded.people_max,source_url=excluded.source_url,source_checked_at=excluded.source_checked_at,active=excluded.active,updated_at=clock_timestamp()`, p.ID, p.Name, p.Activity, p.Kind, p.Area, p.Address, p.Description, p.BudgetMin, p.BudgetMax, p.PriceNote, p.PeopleMin, p.PeopleMax, p.SourceURL, p.SourceCheckedAt, p.Active)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListPlaces(ctx context.Context, f PlaceFilter) (Page[Place], error) {
	page := Page[Place]{Items: []Place{}}
	if f.People == 0 {
		f.People = 1
	}
	if !validPage(f.Cursor, f.Limit) || (f.Activity != "" && !validActivity(f.Activity)) || (f.Kind != "" && f.Kind != "place" && f.Kind != "idea") || utf8.RuneCountInString(f.Area) > 80 || f.People < 1 || f.People > 1000 || (f.MaxBudget != nil && (*f.MaxBudget < 0 || *f.MaxBudget > 1<<31-1)) {
		return page, plans.ErrInvalid
	}
	rows, err := s.db.Query(ctx, `SELECT id::text,name,activity,kind,area,address,description,budget_min,budget_max,price_note,people_min,people_max,source_url,source_checked_at,active FROM places WHERE active AND ($1='' OR activity=$1) AND ($2='' OR position(lower($2) in lower(area))>0) AND ($3::integer IS NULL OR budget_max<=$3) AND people_min<=$4 AND (people_max=0 OR people_max>=$4) AND ($5='' OR id>NULLIF($5,'')::uuid) AND ($6='' OR kind=$6) ORDER BY id LIMIT $7`, f.Activity, strings.TrimSpace(f.Area), f.MaxBudget, f.People, strings.ToLower(f.Cursor), f.Kind, f.Limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Place
		if err = rows.Scan(&p.ID, &p.Name, &p.Activity, &p.Kind, &p.Area, &p.Address, &p.Description, &p.BudgetMin, &p.BudgetMax, &p.PriceNote, &p.PeopleMin, &p.PeopleMax, &p.SourceURL, &p.SourceCheckedAt, &p.Active); err != nil {
			return page, err
		}
		page.Items = append(page.Items, p)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > f.Limit {
		page.Items = page.Items[:f.Limit]
		page.NextCursor = page.Items[f.Limit-1].ID
	}
	return page, nil
}
