package httpapi

import (
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
	"net/http"
	"strconv"
)

func (a *application) places(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	limit, ok := pageLimit(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	filter := store.PlaceFilter{Activity: query.Get("activity"), Kind: query.Get("kind"), Area: query.Get("area"), Cursor: query.Get("cursor"), People: 1, Limit: limit}
	if raw := query.Get("max_budget"); raw != "" {
		budget, err := strconv.Atoi(raw)
		if err != nil {
			apiError(w, r, 400, "invalid_input")
			return
		}
		filter.MaxBudget = &budget
	}
	if raw := query.Get("people"); raw != "" {
		people, err := strconv.Atoi(raw)
		if err != nil || people < 1 {
			apiError(w, r, 400, "invalid_input")
			return
		}
		filter.People = people
	}
	page, err := a.db.ListPlaces(r.Context(), filter)
	if err != nil {
		apiFailure(w, r, err)
		return
	}
	writeJSON(w, 200, page)
}
