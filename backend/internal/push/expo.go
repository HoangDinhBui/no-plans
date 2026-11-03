package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const expoEndpoint = "https://exp.host/--/api/v2/push"

type Expo struct {
	client      *http.Client
	accessToken string
}

func NewExpo(accessToken string) *Expo {
	return &Expo{accessToken: accessToken, client: &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

type Message struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
}

type Result struct {
	Status  string `json:"status"`
	ID      string `json:"id"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

func (e *Expo) Send(ctx context.Context, messages []Message) ([]Result, error) {
	if len(messages) < 1 || len(messages) > 100 {
		return nil, errors.New("Expo send requires 1–100 messages")
	}
	var result []Result
	if err := e.post(ctx, "/send", messages, &result); err != nil {
		return nil, err
	}
	if len(result) != len(messages) {
		return nil, errors.New("Expo ticket count does not match message count")
	}
	for _, ticket := range result {
		if !ticket.valid() || ticket.Status == "ok" && ticket.ID == "" {
			return nil, errors.New("invalid Expo ticket")
		}
	}
	return result, nil
}

func (e *Expo) Receipts(ctx context.Context, ids []string) (map[string]Result, error) {
	if len(ids) < 1 || len(ids) > 1000 {
		return nil, errors.New("Expo receipts require 1–1000 ticket IDs")
	}
	var result map[string]Result
	if err := e.post(ctx, "/getReceipts", map[string][]string{"ids": ids}, &result); err != nil {
		return nil, err
	}
	for _, receipt := range result {
		if !receipt.valid() {
			return nil, errors.New("invalid Expo receipt")
		}
	}
	return result, nil
}

func (r Result) valid() bool {
	return r.Status == "ok" || r.Status == "error" && r.Details.Error != ""
}

type HTTPError struct{ Status int }

func (e *HTTPError) Error() string   { return fmt.Sprintf("Expo HTTP status %d", e.Status) }
func (e *HTTPError) Retryable() bool { return e.Status == 429 || e.Status >= 500 }

func (e *Expo) post(ctx context.Context, path string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoEndpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if e.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.accessToken)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("Expo request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{Status: resp.StatusCode}
	}
	var envelope struct {
		Data   json.RawMessage   `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&envelope); err != nil {
		return errors.New("invalid Expo response")
	}
	if len(envelope.Errors) != 0 || len(envelope.Data) == 0 || bytes.Equal(envelope.Data, []byte("null")) {
		return errors.New("Expo response contains a request error or no data")
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return errors.New("invalid Expo response data")
	}
	return nil
}
