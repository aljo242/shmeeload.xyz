package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// The internal dashboard is pull-only: a red banner is only ever seen by
// someone who happens to open the page. This watches the same roll-up status
// and posts transitions to a Discord webhook, so a problem announces itself.

const (
	alertInterval  = 2 * time.Minute // matches the host push cadence
	alertConfirm   = 2               // consecutive observations before announcing
	alertHeartbeat = 24 * time.Hour  // "still nominal", so silence stays meaningful
	alertMaxLines  = 8               // cap the detail list; the page has the rest
)

// alertState is the transition machine. It holds no IO so it can be tested
// directly, which matters because the interesting behaviour is all in the
// debouncing rather than in the posting.
type alertState struct {
	posted     string // status last announced
	candidate  string // status seen since, not yet confirmed
	candidateN int
	lastPost   time.Time
	started    bool
}

// step folds one observation in and reports whether to post. A status must hold
// for alertConfirm consecutive observations before it is announced, so a single
// scrape landing mid-restart does not page anyone.
func (a *alertState) step(status string, warns, bads []string, now time.Time) (bool, string) {
	// First observation: adopt silently when healthy, announce when not. A
	// process restart should not replay old news, but it should not swallow a
	// problem that is live right now either.
	if !a.started {
		a.started, a.posted, a.lastPost = true, status, now
		if status == statusOK {
			return false, ""
		}
		return true, alertText(status, warns, bads)
	}

	if status == a.posted {
		a.candidate, a.candidateN = "", 0
		if status == statusOK && now.Sub(a.lastPost) >= alertHeartbeat {
			a.lastPost = now
			return true, "[OK] homelab still nominal"
		}
		return false, ""
	}

	if status == a.candidate {
		a.candidateN++
	} else {
		a.candidate, a.candidateN = status, 1
	}
	if a.candidateN < alertConfirm {
		return false, ""
	}
	a.posted, a.candidate, a.candidateN, a.lastPost = status, "", 0, now
	return true, alertText(status, warns, bads)
}

// alertText renders a status plus its detail lines. No emoji anywhere, by house
// rule; the bracketed prefix carries the severity instead.
func alertText(status string, warns, bads []string) string {
	var b strings.Builder
	switch status {
	case statusBad:
		fmt.Fprintf(&b, "[CRITICAL] homelab: %d critical, %d warning", len(bads), len(warns))
	case statusWarn:
		fmt.Fprintf(&b, "[WARN] homelab: %d warning", len(warns))
	default:
		b.WriteString("[OK] homelab recovered, all systems nominal")
	}
	lines := append(append([]string{}, bads...), warns...)
	for i, l := range lines {
		if i == alertMaxLines {
			fmt.Fprintf(&b, "\n... and %d more", len(lines)-alertMaxLines)
			break
		}
		b.WriteString("\n- " + l)
	}
	return b.String()
}

// postDiscord sends one webhook message. Failures are logged and dropped: an
// alerter that blocks or crashes the server would be worse than the gap it
// fills.
func postDiscord(ctx context.Context, client *http.Client, webhook, msg string) {
	body, err := json.Marshal(map[string]string{"content": msg})
	if err != nil {
		log.Printf("alert: marshal: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rq, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		log.Printf("alert: request: %v", err)
		return
	}
	rq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(rq)
	if err != nil {
		log.Printf("alert: post: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		log.Printf("alert: webhook returned %s", resp.Status)
	}
}

// runAlerter polls the roll-up on a ticker and posts confirmed transitions. snap
// returns the current status and its detail lines, which lets main wire in
// buildInternalAPI without this file needing every dependency it takes.
func runAlerter(ctx context.Context, webhook string, snap func() (string, []string, []string)) {
	if webhook == "" {
		return
	}
	client := &http.Client{Timeout: 15 * time.Second}
	st := &alertState{}
	t := time.NewTicker(alertInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			status, warns, bads := snap()
			if post, msg := st.step(status, warns, bads, now); post {
				postDiscord(ctx, client, webhook, msg)
			}
		}
	}
}
