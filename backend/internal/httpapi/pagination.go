package httpapi

import (
	"net/http"
	"strconv"
)

func pageLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		var err error
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			apiError(w, r, 400, "invalid_input")
			return 0, false
		}
	}
	return limit, true
}
