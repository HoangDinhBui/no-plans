package push

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/HoangDinhBui/no-plans/backend/internal/store"
)

func TestDeliveryRetriesTransientHTTPFailures(t *testing.T) {
	for _, ticket := range []string{"", "ticket"} {
		for _, status := range []int{400, 429, 503} {
			client := NewExpo("")
			client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("{}"))}, nil
			})
			_, failure, delay := processDelivery(context.Background(), client, store.PushDelivery{TicketID: ticket, Attempt: 1})
			if failure == "" || (delay != nil) != (status != 400) {
				t.Errorf("ticket %q, HTTP %d: failure=%q, retry=%v", ticket, status, failure, delay)
			}
		}
	}
}
