package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// domainCount is one entry in the top-blocked list.
type domainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// piholeStats is the snapshot the internal view renders. Up is false when the
// last poll could not reach or authenticate to Pi-hole.
type piholeStats struct {
	Up             bool          `json:"up"`
	QueriesToday   int           `json:"queriesToday"`
	BlockedToday   int           `json:"blockedToday"`
	PercentBlocked float64       `json:"percentBlocked"`
	BlocklistSize  int           `json:"blocklistSize"`
	ActiveClients  int           `json:"activeClients"`
	TopBlocked     []domainCount `json:"topBlocked"`
	CheckedAt      int64         `json:"checkedAt"` // unix seconds
}

// piholeClient polls the Pi-hole v6 API and hands out the last snapshot. It
// reuses a session id until it expires, re-authenticating only when needed so
// it does not exhaust Pi-hole's limited API sessions.
type piholeClient struct {
	baseURL  string
	password string
	client   *http.Client

	authMu sync.Mutex
	sid    string
	sidExp time.Time

	snapMu sync.RWMutex
	snap   piholeStats
}

func newPiholeClient(baseURL, password string) *piholeClient {
	c := &piholeClient{
		baseURL:  baseURL,
		password: password,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
	if baseURL != "" && password != "" {
		go c.loop()
	}
	return c
}

func (c *piholeClient) loop() {
	c.refresh()
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for range t.C {
		c.refresh()
	}
}

func (c *piholeClient) snapshot() piholeStats {
	c.snapMu.RLock()
	defer c.snapMu.RUnlock()
	return c.snap
}

// refresh fetches a fresh snapshot, marking Pi-hole down on any failure.
func (c *piholeClient) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.fetch(ctx)
	if err != nil {
		stats = piholeStats{Up: false}
	}
	stats.CheckedAt = time.Now().Unix()

	c.snapMu.Lock()
	c.snap = stats
	c.snapMu.Unlock()
}

func (c *piholeClient) fetch(ctx context.Context) (piholeStats, error) {
	var out piholeStats

	var summary struct {
		Queries struct {
			Total          int     `json:"total"`
			Blocked        int     `json:"blocked"`
			PercentBlocked float64 `json:"percent_blocked"`
		} `json:"queries"`
		Gravity struct {
			DomainsBeingBlocked int `json:"domains_being_blocked"`
		} `json:"gravity"`
		Clients struct {
			Active int `json:"active"`
		} `json:"clients"`
	}
	if err := c.getJSON(ctx, "/api/stats/summary", &summary); err != nil {
		return out, err
	}

	var top struct {
		Domains []domainCount `json:"domains"`
	}
	if err := c.getJSON(ctx, "/api/stats/top_domains?blocked=true&count=8", &top); err != nil {
		return out, err
	}

	out = piholeStats{
		Up:             true,
		QueriesToday:   summary.Queries.Total,
		BlockedToday:   summary.Queries.Blocked,
		PercentBlocked: summary.Queries.PercentBlocked,
		BlocklistSize:  summary.Gravity.DomainsBeingBlocked,
		ActiveClients:  summary.Clients.Active,
		TopBlocked:     top.Domains,
	}
	return out, nil
}

// getJSON does an authenticated GET, re-authenticating once if the session has
// expired (Pi-hole answers 401).
func (c *piholeClient) getJSON(ctx context.Context, path string, dst any) error {
	sid, err := c.ensureSID(ctx, false)
	if err != nil {
		return err
	}
	status, body, err := c.doGet(ctx, path, sid)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized {
		sid, err = c.ensureSID(ctx, true) // force re-auth
		if err != nil {
			return err
		}
		if status, body, err = c.doGet(ctx, path, sid); err != nil {
			return err
		}
	}
	if status != http.StatusOK {
		return fmt.Errorf("pihole %s: status %d", path, status)
	}
	return json.Unmarshal(body, dst)
}

func (c *piholeClient) doGet(ctx context.Context, path, sid string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Ftl-Sid", sid) // Go canonicalizes anyway; Pi-hole matches case-insensitively
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, body, err
}

// ensureSID returns a valid session id, authenticating when none is cached, the
// cached one has expired, or force is set.
func (c *piholeClient) ensureSID(ctx context.Context, force bool) (string, error) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if !force && c.sid != "" && time.Now().Before(c.sidExp) {
		return c.sid, nil
	}

	reqBody, _ := json.Marshal(map[string]string{"password": c.password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", err
	}
	var auth struct {
		Session struct {
			Valid    bool   `json:"valid"`
			SID      string `json:"sid"`
			Validity int    `json:"validity"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &auth); err != nil {
		return "", err
	}
	if !auth.Session.Valid || auth.Session.SID == "" {
		return "", fmt.Errorf("pihole auth failed: status %d", resp.StatusCode)
	}
	c.sid = auth.Session.SID
	// Refresh a minute before the session actually lapses.
	c.sidExp = time.Now().Add(time.Duration(auth.Session.Validity)*time.Second - time.Minute)
	return c.sid, nil
}
