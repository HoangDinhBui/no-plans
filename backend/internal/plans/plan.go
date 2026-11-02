// Package plans defines business rules. Callers must lock the persisted plan
// before applying mutations and atomically save the resulting outbox events.
package plans

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid_input")
	ErrFull      = errors.New("plan_full")
	ErrClosed    = errors.New("plan_closed")
	ErrConflict  = errors.New("version_conflict")
)

type Plan struct {
	ID               string            `json:"id"`
	OwnerID          string            `json:"owner_id"`
	GroupID          string            `json:"group_id,omitempty"`
	SelectedOptionID string            `json:"selected_option_id,omitempty"`
	Title            string            `json:"title"`
	Activity         string            `json:"activity"`
	StartsAt         time.Time         `json:"starts_at"`
	Capacity         int               `json:"capacity"` // Zero means unlimited.
	Status           string            `json:"status"`
	Location         string            `json:"location"`
	Version          int64             `json:"version"`
	Members          map[string]string `json:"members"`
}

func New(id, owner, title, activity string, startsAt time.Time, capacity int, now time.Time) (*Plan, error) {
	title = strings.TrimSpace(title)
	if id == "" || owner == "" || !validDetails(title, startsAt, capacity, now) {
		return nil, ErrInvalid
	}
	switch activity {
	case "food", "coffee", "drinks", "cinema", "sports", "other":
	default:
		return nil, ErrInvalid
	}
	return &Plan{ID: id, OwnerID: owner, Title: title, Activity: activity, StartsAt: startsAt.UTC(), Capacity: capacity, Status: "inviting", Version: 1, Members: map[string]string{owner: "going"}}, nil
}

func validDetails(title string, at time.Time, capacity int, now time.Time) bool {
	return strings.TrimSpace(title) != "" && utf8.RuneCountInString(title) <= 120 && at.After(now) && capacity >= 0 && capacity <= 1<<31-1
}
func (p *Plan) closed() bool { return p.Status != "inviting" && p.Status != "confirmed" }
func (p *Plan) Going() int {
	count := 0
	for _, r := range p.Members {
		if r == "going" {
			count++
		}
	}
	return count
}

// Respond returns false for a retry with no state change (and no new event).
func (p *Plan) Respond(actorID, response string, now time.Time) (bool, error) {
	previous, member := p.Members[actorID]
	if !member {
		return false, ErrForbidden
	}
	if response != "going" && response != "not_going" && response != "maybe" {
		return false, ErrInvalid
	}
	if p.closed() || !p.StartsAt.After(now) {
		return false, ErrClosed
	}
	if previous == response {
		return false, nil
	}
	if response == "going" && p.Capacity > 0 && p.Going() >= p.Capacity {
		return false, ErrFull
	}
	p.Members[actorID] = response
	p.Version++
	return true, nil
}

func (p *Plan) Transition(actorID, action string, now time.Time) error {
	if actorID != p.OwnerID {
		return ErrForbidden
	}
	if p.closed() {
		return ErrClosed
	}
	switch action {
	case "confirm":
		if p.Status != "inviting" || strings.TrimSpace(p.Location) == "" || !p.StartsAt.After(now) {
			return ErrInvalid
		}
		p.Status = "confirmed"
	case "cancel":
		p.Status = "cancelled"
	case "complete":
		if p.Status != "confirmed" || now.Before(p.StartsAt) {
			return ErrInvalid
		}
		p.Status = "completed"
	default:
		return ErrInvalid
	}
	p.Version++
	return nil
}

func (p *Plan) Edit(actorID string, expectedVersion int64, title string, startsAt time.Time, capacity int, location string, now time.Time) error {
	if actorID != p.OwnerID {
		return ErrForbidden
	}
	if p.closed() {
		return ErrClosed
	}
	if p.Version != expectedVersion {
		return ErrConflict
	}
	title = strings.TrimSpace(title)
	location = strings.TrimSpace(location)
	if !validDetails(title, startsAt, capacity, now) || utf8.RuneCountInString(location) > 500 {
		return ErrInvalid
	}
	if capacity > 0 && capacity < p.Going() {
		return ErrFull
	}
	if p.Status == "confirmed" && location == "" {
		return ErrInvalid
	}
	p.Title = title
	if p.Location != location {
		p.SelectedOptionID = ""
	}
	p.StartsAt = startsAt.UTC()
	p.Capacity = capacity
	p.Location = location
	p.Version++
	return nil
}
