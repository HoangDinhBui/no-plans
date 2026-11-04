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

func TestDeliverySchedulesReceiptAfterTicket(t *testing.T) {
	client := NewExpo("")
	client.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"data":[{"status":"ok","id":"ticket"}]}`
		if strings.HasSuffix(r.URL.Path, "/getReceipts") {
			body = `{"data":{"ticket":{"status":"ok"}}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	delivery := store.PushDelivery{Token: "ExpoPushToken[fixture]", Attempt: 1}
	ticket, failure, delay := processDelivery(context.Background(), client, delivery)
	if ticket != "ticket" || failure != "" || delay == nil || delay.Minutes() != 15 {
		t.Fatalf("send result: %q %q %v", ticket, failure, delay)
	}
	delivery.TicketID = ticket
	delivery.Attempt++
	ticket, failure, delay = processDelivery(context.Background(), client, delivery)
	if ticket != "" || failure != "" || delay != nil {
		t.Fatalf("receipt result: %q %q %v", ticket, failure, delay)
	}
}
