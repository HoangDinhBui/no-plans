package push

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestExpoTicketAndReceipt(t *testing.T) {
	client := NewExpo("test-access")
	client.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-access" {
			t.Fatalf("unexpected request method or authorization")
		}
		var body string
		switch r.URL.String() {
		case expoEndpoint + "/send":
			body = `{"data":[{"status":"ok","id":"ticket"}]}`
		case expoEndpoint + "/getReceipts":
			body = `{"data":{"ticket":{"status":"error","details":{"error":"DeviceNotRegistered"}}}}`
		default:
			t.Fatal("unexpected endpoint")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	tickets, err := client.Send(context.Background(), []Message{{To: "ExpoPushToken[test]", Title: "No Plans"}})
	if err != nil || len(tickets) != 1 || tickets[0].ID != "ticket" {
		t.Fatalf("tickets: %+v, %v", tickets, err)
	}
	receipts, err := client.Receipts(context.Background(), []string{tickets[0].ID})
	if err != nil || receipts["ticket"].Details.Error != "DeviceNotRegistered" {
		t.Fatalf("receipts: %+v, %v", receipts, err)
	}
}
