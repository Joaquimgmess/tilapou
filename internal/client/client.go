package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrDaemonUnreachable = errors.New("client: daemon nao respondeu")
	ErrRequestFailed     = errors.New("client: o daemon recusou a chamada")
)

type Tank struct {
	ID           uint32    `json:"id"`
	Kind         string    `json:"kind"`
	Fish         int32     `json:"fish"`
	MeanGrams    int64     `json:"mean_grams"`
	FeedKg       int64     `json:"feed_kg"`
	OxygenUgL    int32     `json:"oxygen_ugl"`
	Aerating     bool      `json:"aerating"`
	DensityMilli int64     `json:"density_milli_kg_m3"`
	Ready        bool      `json:"ready_to_harvest"`
	BatchID      uint32    `json:"batch_id"`
	PriceKgCents int64     `json:"price_kg_cents"`
	NextClassG   int64     `json:"next_class_grams"`
	Sick         bool      `json:"sick"`
	Capacity     int64     `json:"capacity_fish"`
	ServedFor    int64     `json:"served_for_ticks"`
	Upgrades     []Upgrade `json:"upgrades"`
}

type Event struct {
	Seq    uint64 `json:"seq"`
	Kind   string `json:"kind"`
	From   int64  `json:"from_tick"`
	To     int64  `json:"to_tick"`
	Tank   uint32 `json:"tank_id"`
	Fish   int32  `json:"fish"`
	MassG  int64  `json:"mass_grams"`
	CashTC int64  `json:"cash_cents"`
	Reason string `json:"reason"`
}

type Upgrade struct {
	Kind      string `json:"kind"`
	Owned     bool   `json:"owned"`
	CostCents int64  `json:"cost_cents"`
}

type Outcome struct {
	Applied    bool   `json:"applied"`
	Reason     string `json:"reason"`
	NeededCash int64  `json:"needed_cents"`
}

type Prices struct {
	FeedKgCents     int64 `json:"feed_kg_cents"`
	FingerlingCents int64 `json:"fingerling_cents"`
	FishKgCents     int64 `json:"fish_kg_cents"`
	RatioPPM        int64 `json:"equivalence_ppm"`
	ViablePPM       int64 `json:"equivalence_viable_ppm"`
}

type Cycle struct {
	Fish       int32 `json:"fish"`
	MassG      int64 `json:"mass_grams"`
	RevenueTC  int64 `json:"revenue_cents"`
	CostTC     int64 `json:"cost_cents"`
	MarginTC   int64 `json:"margin_cents"`
	CostPerKg  int64 `json:"cost_per_kg_cents"`
	PricePerKg int64 `json:"price_per_kg_cents"`
	FCRPPM     int64 `json:"fcr_ppm"`
}

type Snapshot struct {
	FarmID      string   `json:"farm_id"`
	Name        string   `json:"name"`
	Tick        int64    `json:"tick"`
	Hour        int32    `json:"hour"`
	TempMilliC  int32    `json:"temp_milli_c"`
	CashCents   int64    `json:"cash_cents"`
	LifetimeTC  int64    `json:"lifetime_cents"`
	BiomassG    int64    `json:"biomass_grams"`
	Fish        int32    `json:"fish"`
	Prestige    uint32   `json:"prestige"`
	Tanks       []Tank   `json:"tanks"`
	PrestigeNow uint32   `json:"prestige_available"`
	Prices      Prices   `json:"prices"`
	Debt        int64    `json:"debt_cents"`
	LastCycle   Cycle    `json:"last_cycle"`
	Events      []Event  `json:"events"`
	LastApplied bool     `json:"last_action_applied"`
	LastReason  string   `json:"last_action_reason"`
	LastOutcome *Outcome `json:"last_outcome,omitempty"`
}

type Action struct {
	Key      uint64 `json:"key"`
	Kind     string `json:"kind"`
	Tank     uint32 `json:"tank_id,omitempty"`
	Batch    uint32 `json:"batch_id,omitempty"`
	TankKind string `json:"tank_kind,omitempty"`
	Auto     string `json:"auto,omitempty"`
	Amount   int64  `json:"amount,omitempty"`
}

type Client struct {
	base string
	http *http.Client
}

func New(base string, timeout time.Duration) *Client {
	return &Client{base: base, http: &http.Client{Timeout: timeout}}
}

func (c *Client) Snapshot(ctx context.Context) (Snapshot, error) {
	return c.do(ctx, http.MethodGet, "/v1/farm", nil)
}

func (c *Client) Act(ctx context.Context, action Action) (Snapshot, error) {
	body, err := json.Marshal(action)
	if err != nil {
		return Snapshot{}, fmt.Errorf("client: encoding action: %w", err)
	}

	return c.do(ctx, http.MethodPost, "/v1/farm/actions", body)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (Snapshot, error) {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return Snapshot{}, fmt.Errorf("client: building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: %w", ErrDaemonUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return Snapshot{}, problemOf(resp)
	}

	var snapshot Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("client: decoding snapshot: %w", err)
	}

	return snapshot, nil
}

type problem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

func problemOf(resp *http.Response) error {
	var p problem
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
	}

	return fmt.Errorf("%w: %s: %s", ErrRequestFailed, p.Title, p.Detail)
}
