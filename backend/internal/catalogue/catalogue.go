package catalogue

import (
	_ "embed"
	"encoding/json"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

//go:embed initial.json
var initial []byte

func Initial() ([]store.Place, error) {
	var places []store.Place
	err := json.Unmarshal(initial, &places)
	return places, err
}
