package httpapi

import (
	"context"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/catalogue"
	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

func TestBundledCatalogueFeedsFoodAndPlaceDiscovery(t *testing.T) {
	f := setupApp(t)
	places, err := catalogue.Initial()
	if err != nil {
		t.Fatal(err)
	}
	if err = f.db.UpsertPlaces(context.Background(), places); err != nil {
		t.Fatal(err)
	}
	if err = f.db.UpsertPlaces(context.Background(), places); err != nil {
		t.Fatal(err)
	}
	token := f.token(ids.New())
	for _, path := range []string{"/v1/places?activity=food&max_budget=80000&people=2", "/v1/places?kind=place"} {
		page := call(t, f.handler, "GET", path, token, "", "", 200)
		if len(page["items"].([]any)) == 0 {
			t.Fatalf("discovery is empty: %s", path)
		}
	}
}
