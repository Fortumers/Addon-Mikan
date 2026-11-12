package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Panel is the host API: key-value storage and the Bot API proxy.
// The real Telegram token never reaches this process.
type Panel struct {
	BaseURL  string
	Token    string
	BotURL   string
	BotToken string
	HTTP     *http.Client
}

func (p *Panel) client() *http.Client {
	if p != nil && p.HTTP != nil {
		return p.HTTP
	}
	return &http.Client{Timeout: 45 * time.Second}
}

func (p *Panel) enabled() bool {
	return p != nil && p.BaseURL != "" && p.Token != ""
}

func (p *Panel) botEnabled() bool {
	return p != nil && p.BotURL != "" && p.BotToken != ""
}

var errNotFound = errors.New("not found")

func (p *Panel) kvGet(ctx context.Context, key string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.BaseURL, "/")+"/kv/"+key, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	resp, err := p.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16+1))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("kv get %s: %s", key, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (p *Panel) kvPut(ctx context.Context, key string, value []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(p.BaseURL, "/")+"/kv/"+key, bytes.NewReader(value))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("kv put %s: %s", key, strings.TrimSpace(string(body)))
	}
	return nil
}

type tgEnvelope struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func (p *Panel) tg(ctx context.Context, method string, payload any, dest any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	u := strings.TrimRight(p.BotURL, "/") + "/bot" + p.BotToken + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return err
	}
	var env tgEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("telegram %s: %s", method, strings.TrimSpace(string(raw)))
	}
	if !env.OK {
		if env.Description == "" {
			env.Description = resp.Status
		}
		return fmt.Errorf("telegram %s: %s", method, env.Description)
	}
	if dest != nil && len(env.Result) > 0 && string(env.Result) != "null" {
		return json.Unmarshal(env.Result, dest)
	}
	return nil
}
