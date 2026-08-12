package main

import (
	"strings"
	"testing"
	"time"
)

func TestAlertStateFirstObservation(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)

	// Healthy on startup is adopted silently: a restart should not replay news.
	var ok alertState
	if post, _ := ok.step("ok", nil, nil, now); post {
		t.Fatal("healthy first observation should not post")
	}

	// Unhealthy on startup does post, because the problem is live right now.
	var bad alertState
	post, msg := bad.step("bad", nil, []string{"foundry: valheim down"}, now)
	if !post {
		t.Fatal("unhealthy first observation should post")
	}
	if !strings.Contains(msg, "valheim down") {
		t.Fatalf("message should carry the detail, got %q", msg)
	}
}

func TestAlertStateDebouncesBlips(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	var a alertState
	a.step("ok", nil, nil, now)

	// One bad observation is not enough; a scrape landing mid-restart is common.
	now = now.Add(alertInterval)
	if post, _ := a.step("bad", nil, []string{"x"}, now); post {
		t.Fatal("a single bad observation should not post")
	}

	// Recovering before confirmation means nothing is ever announced.
	now = now.Add(alertInterval)
	if post, _ := a.step("ok", nil, nil, now); post {
		t.Fatal("recovery before confirmation should not post")
	}
	if a.posted != "ok" {
		t.Fatalf("posted status should still be ok, got %q", a.posted)
	}
}

func TestAlertStateConfirmsAndRecovers(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	var a alertState
	a.step("ok", nil, nil, now)

	// Sustained for alertConfirm observations, so it is announced.
	var post bool
	var msg string
	for i := 0; i < alertConfirm; i++ {
		now = now.Add(alertInterval)
		post, msg = a.step("bad", []string{"w"}, []string{"c"}, now)
	}
	if !post {
		t.Fatal("a sustained bad status should post")
	}
	if !strings.HasPrefix(msg, "[CRITICAL]") {
		t.Fatalf("bad status should be prefixed [CRITICAL], got %q", msg)
	}

	// Already-announced status must not repeat every tick.
	now = now.Add(alertInterval)
	if post, _ = a.step("bad", []string{"w"}, []string{"c"}, now); post {
		t.Fatal("an already-announced status should not repeat")
	}

	// Recovery is worth announcing too, otherwise the last word is the failure.
	for i := 0; i < alertConfirm; i++ {
		now = now.Add(alertInterval)
		post, msg = a.step("ok", nil, nil, now)
	}
	if !post {
		t.Fatal("recovery should post")
	}
	if !strings.Contains(msg, "recovered") {
		t.Fatalf("recovery message should say so, got %q", msg)
	}
}

func TestAlertStateHeartbeat(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	var a alertState
	a.step("ok", nil, nil, now)

	// Quiet for less than a day stays quiet.
	if post, _ := a.step("ok", nil, nil, now.Add(alertHeartbeat-time.Minute)); post {
		t.Fatal("heartbeat should not fire early")
	}
	// Past a day, one "still nominal" so continued silence stays meaningful.
	post, msg := a.step("ok", nil, nil, now.Add(alertHeartbeat))
	if !post {
		t.Fatal("heartbeat should fire after the interval")
	}
	if !strings.Contains(msg, "still nominal") {
		t.Fatalf("unexpected heartbeat text %q", msg)
	}
}

func TestAlertTextTruncatesDetail(t *testing.T) {
	t.Parallel()
	bads := make([]string, alertMaxLines+3)
	for i := range bads {
		bads[i] = "problem"
	}
	msg := alertText("bad", nil, bads)
	if !strings.Contains(msg, "and 3 more") {
		t.Fatalf("long detail lists should be truncated, got %q", msg)
	}
}

func TestNoEmojiInAlerts(t *testing.T) {
	t.Parallel()
	// House rule: no emoji anywhere, including machine-generated messages.
	for _, msg := range []string{
		alertText("bad", []string{"w"}, []string{"c"}),
		alertText("warn", []string{"w"}, nil),
		alertText("ok", nil, nil),
	} {
		for _, r := range msg {
			if r > 0x2100 {
				t.Fatalf("alert text contains a non-ASCII symbol %q in %q", r, msg)
			}
		}
	}
}
