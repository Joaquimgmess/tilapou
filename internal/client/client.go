// Package client is the HTTP transport for the tilapou daemon.
//
// Os tipos do contrato moram em internal/api e sao os mesmos que o daemon escreve: espelhar
// aqui deixava campo novo compilar de um lado so e nunca chegar a tela.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Joaquimgmess/tilapou/internal/api"
)

var (
	// ErrDaemonUnreachable reports an incomplete HTTP call: daemon down, DNS,
	// connection refused, timeout or cancelled context.
	ErrDaemonUnreachable = errors.New("client: daemon nao respondeu")
	// ErrRequestFailed reports a status other than 200 or 201, with the
	// problem+json detail wrapped in the message.
	ErrRequestFailed = errors.New("client: o daemon recusou a chamada")
)

// Action is the command sent to the daemon; Key is the idempotency key.
type Action struct {
	Key      uint64 `json:"key"`
	Kind     string `json:"kind"`
	Tank     uint32 `json:"tank_id,omitempty"`
	Batch    uint32 `json:"batch_id,omitempty"`
	TankKind string `json:"tank_kind,omitempty"`
	Auto     string `json:"auto,omitempty"`
	Amount   int64  `json:"amount,omitempty"`
	// SeenTick e o tick que estava na tela quando o jogador decidiu: o daemon recusa a acao
	// que chega contra um mundo que ja andou demais.
	SeenTick int64 `json:"seen_tick,omitempty"`
}

// Client talks to the tilapou daemon over HTTP.
type Client struct {
	base string
	http *http.Client
}

// New creates a Client with a per-request timeout.
func New(base string, timeout time.Duration) *Client {
	return &Client{base: base, http: &http.Client{Timeout: timeout}}
}

// Snapshot performs GET /v1/farm.
func (c *Client) Snapshot(ctx context.Context) (api.Snapshot, error) {
	return c.do(ctx, http.MethodGet, "/v1/farm", nil)
}

// Act performs POST /v1/farm/actions.
func (c *Client) Act(ctx context.Context, action Action) (api.Snapshot, error) {
	body, err := json.Marshal(action)
	if err != nil {
		return api.Snapshot{}, fmt.Errorf("client: encoding action: %w", err)
	}

	return c.do(ctx, http.MethodPost, "/v1/farm/actions", body)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (api.Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return api.Snapshot{}, fmt.Errorf("client: building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return api.Snapshot{}, fmt.Errorf("%w: %w", ErrDaemonUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return api.Snapshot{}, problemOf(resp)
	}

	var snapshot api.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return api.Snapshot{}, fmt.Errorf("client: decoding snapshot: %w", err)
	}

	return snapshot, nil
}

type problem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func problemOf(resp *http.Response) error {
	var p problem
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	return fmt.Errorf("%w: %s: %s", ErrRequestFailed, p.Title, p.Detail)
}
