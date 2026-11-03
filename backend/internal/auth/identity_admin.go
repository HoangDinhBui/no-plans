package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
)

type IdentityAdmin struct {
	base   string
	key    string
	client *http.Client
}

func NewIdentityAdmin(projectURL, key string) (*IdentityAdmin, error) {
	u, err := url.Parse(projectURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || strings.TrimSpace(key) == "" {
		return nil, errors.New("invalid identity admin configuration")
	}
	return &IdentityAdmin{
		base: strings.TrimRight(u.String(), "/"), key: key,
		client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func (a *IdentityAdmin) DeleteUser(ctx context.Context, id string) error {
	if !ids.Valid(id) {
		return errors.New("invalid identity id")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, a.base+"/auth/v1/admin/users/"+id, strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", a.key)
	req.Header.Set("Authorization", "Bearer "+a.key)
	req.Header.Set("Content-Type", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		return errors.New("identity deletion request failed")
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	if res.StatusCode == http.StatusNotFound {
		var body struct {
			Code string `json:"error_code"`
		}
		if json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body) == nil && body.Code == "user_not_found" {
			return nil
		}
	}
	return fmt.Errorf("identity deletion returned HTTP %d", res.StatusCode)
}
