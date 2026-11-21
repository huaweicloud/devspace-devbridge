/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestDevbridge() *Devbridge {
	return &Devbridge{
		apiKey:       "test-api-key",
		logger:       slog.Default(),
		outputWriter: &bytes.Buffer{},
	}
}

func TestParsePortNotification(t *testing.T) {
	cases := []struct {
		name    string
		data    string
		wantOK  bool
		wantNil bool
	}{
		{"relay request notification", `{"@type":"RelayRequest","ports":[8080,3000]}`, true, false},
		{"empty ports notification", `{"@type":"RelayRequest","ports":[]}`, true, true},
		{"ssh banner is not a notification", "SSH-2.0-devtunnels_1.0\r\n", false, false},
		{"other json type", `{"@type":"Data"}`, false, false},
		{"invalid json", "not json at all", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ports, ok := parsePortNotification([]byte(tc.data))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if tc.wantOK && tc.wantNil && len(ports) != 0 {
				t.Fatalf("expected empty ports, got %v", ports)
			}
			if tc.wantOK && !tc.wantNil {
				if len(ports) != 2 || ports[0] != 8080 || ports[1] != 3000 {
					t.Fatalf("unexpected ports %v", ports)
				}
			}
		})
	}
}

func TestPortNotifierUpdate(t *testing.T) {
	pn := newPortNotifier()

	added, removed, changed := pn.update([]int{8080})
	if added != nil || removed != nil || changed {
		t.Fatalf("first notification must not report changes, got added=%v removed=%v changed=%v", added, removed, changed)
	}
	select {
	case <-pn.ready:
	default:
		t.Fatal("first notification must close ready")
	}
	if got := pn.currentPorts(); len(got) != 1 || got[0] != 8080 {
		t.Fatalf("currentPorts = %v, want [8080]", got)
	}

	added, removed, changed = pn.update([]int{8080, 3000})
	if !changed {
		t.Fatal("second notification adding a port must report changed")
	}
	if len(added) != 1 || added[0] != 3000 {
		t.Fatalf("added = %v, want [3000]", added)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %v, want empty", removed)
	}

	added, removed, changed = pn.update([]int{3000})
	if !changed {
		t.Fatal("notification removing a port must report changed")
	}
	if len(removed) != 1 || removed[0] != 8080 {
		t.Fatalf("removed = %v, want [8080]", removed)
	}

	added, removed, changed = pn.update([]int{3000})
	if changed {
		t.Fatal("identical port set must be idempotent (no change)")
	}

	select {
	case <-pn.ready:
	default:
		t.Fatal("ready must stay closed after repeated updates")
	}
}

// fakePortLister is a controllable ListPorts replacement.
type fakePortLister struct {
	mu     sync.Mutex
	ports  []Port
	err    error
	calls  int
	block  time.Duration
	cancel bool
}

func (f *fakePortLister) list(ctx context.Context, _ string) ([]Port, error) {
	f.mu.Lock()
	f.calls++
	ports, err, block, cancel := f.ports, f.err, f.block, f.cancel
	f.mu.Unlock()

	if block > 0 {
		select {
		case <-time.After(block):
		case <-ctx.Done():
		}
	}
	if cancel {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	out := make([]Port, len(ports))
	copy(out, ports)
	return out, nil
}

func (f *fakePortLister) set(ports []int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ports = make([]Port, len(ports))
	for i, p := range ports {
		f.ports[i] = Port{Port: p}
	}
}

func (f *fakePortLister) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestRefresher(t *testing.T, lister *fakePortLister, static []int, pn *portNotifier, onChange func(added, removed, current []int)) (*portSetRefresher, *fakeClock) {
	t.Helper()
	d := newTestDevbridge()
	r := newPortSetRefresher(d, "abcd2345", static, pn, onChange)
	clk := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	r.now = clk.Now
	if lister != nil {
		r.listPorts = lister.list
	}
	return r, clk
}

// fakeClock advances deterministically so debounce assertions never depend on
// wall-clock timing.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestPortSetRefresherAppliesRuntimeChanges(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080})
	var changes []struct{ added, removed, current []int }
	var callbacks int
	r, clk := newTestRefresher(t, lister, nil, newPortNotifier(), func(added, removed, current []int) {
		callbacks++
		changes = append(changes, struct{ added, removed, current []int }{added, removed, current})
	})

	ports, fresh := r.Refresh(context.Background(), true)
	if !fresh {
		t.Fatal("first refresh must report a fresh API result")
	}
	if len(ports) != 1 || ports[0] != 8080 {
		t.Fatalf("ports = %v, want [8080]", ports)
	}
	if callbacks != 0 {
		t.Fatal("initial port set must not trigger OnPortsChanged")
	}

	// Simulate `devbridge port create <tunnel> -p 18081` while the Host runs.
	lister.set([]int{8080, 18081})
	clk.Advance(2 * time.Second)
	ports, fresh = r.Refresh(context.Background(), false)
	if !fresh {
		t.Fatal("refresh after a port create must report a fresh API result")
	}
	if len(ports) != 2 || ports[0] != 8080 || ports[1] != 18081 {
		t.Fatalf("ports = %v, want [8080 18081]", ports)
	}
	if callbacks != 1 {
		t.Fatalf("OnPortsChanged calls = %d, want 1", callbacks)
	}
	if len(changes[0].added) != 1 || changes[0].added[0] != 18081 {
		t.Fatalf("added = %v, want [18081]", changes[0].added)
	}
	if len(changes[0].removed) != 0 {
		t.Fatalf("removed = %v, want empty", changes[0].removed)
	}
	if len(changes[0].current) != 2 {
		t.Fatalf("current = %v, want 2 ports", changes[0].current)
	}
}

func TestPortSetRefresherIsIdempotent(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080, 18081})
	var callbacks int
	r, clk := newTestRefresher(t, lister, nil, newPortNotifier(), func(added, removed, current []int) { callbacks++ })

	if _, fresh := r.Refresh(context.Background(), true); !fresh {
		t.Fatal("first refresh must report a fresh API result")
	}
	for i := 0; i < 3; i++ {
		clk.Advance(2 * time.Second)
		ports, fresh := r.Refresh(context.Background(), false)
		if !fresh {
			t.Fatalf("refresh %d must report a fresh API result", i)
		}
		if len(ports) != 2 {
			t.Fatalf("refresh %d ports = %v, want 2 ports", i, ports)
		}
	}
	if callbacks != 0 {
		t.Fatalf("an unchanged port set must never trigger OnPortsChanged, calls = %d", callbacks)
	}
	if got := lister.callCount(); got != 4 {
		t.Fatalf("ListPorts calls = %d, want 4", got)
	}
}

func TestPortSetRefresherRemovesPorts(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080, 18081})
	var removed []int
	var current []int
	r, clk := newTestRefresher(t, lister, nil, newPortNotifier(), func(added, gone, cur []int) {
		removed = gone
		current = cur
	})

	if _, fresh := r.Refresh(context.Background(), true); !fresh {
		t.Fatal("first refresh must report a fresh API result")
	}

	// Simulate `devbridge port delete <tunnel> -p 18081`.
	lister.set([]int{8080})
	clk.Advance(2 * time.Second)
	ports, _ := r.Refresh(context.Background(), false)
	if len(ports) != 1 || ports[0] != 8080 {
		t.Fatalf("ports = %v, want [8080]", ports)
	}
	if len(removed) != 1 || removed[0] != 18081 {
		t.Fatalf("removed = %v, want [18081]", removed)
	}
	if len(current) != 1 || current[0] != 8080 {
		t.Fatalf("current = %v, want [8080]", current)
	}
}

func TestPortSetRefresherStaticModeIgnoresAPIChanges(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080, 18081})
	var callbacks int
	r, _ := newTestRefresher(t, lister, []int{8080}, newPortNotifier(), func(added, removed, current []int) { callbacks++ })

	for i := 0; i < 2; i++ {
		ports, fresh := r.Refresh(context.Background(), true)
		if fresh {
			t.Fatal("static ports mode must never report a fresh API result")
		}
		if len(ports) != 1 || ports[0] != 8080 {
			t.Fatalf("static ports mode must keep pinned ports, got %v", ports)
		}
	}
	if callbacks != 0 {
		t.Fatalf("static ports mode must not trigger OnPortsChanged, calls = %d", callbacks)
	}
	if got := lister.callCount(); got != 0 {
		t.Fatalf("static ports mode must not call ListPorts, calls = %d", got)
	}
}

func TestPortSetRefresherDebouncesRapidChannelAccepts(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080})
	r, clk := newTestRefresher(t, lister, nil, newPortNotifier(), nil)

	if _, _ = r.Refresh(context.Background(), true); lister.callCount() != 1 {
		t.Fatalf("ListPorts calls = %d, want 1", lister.callCount())
	}
	clk.Advance(200 * time.Millisecond)
	if _, fresh := r.Refresh(context.Background(), false); !fresh {
		t.Fatal("debounced refresh must still use a fresh (cached) port set")
	}
	if got := lister.callCount(); got != 1 {
		t.Fatalf("ListPorts calls inside the debounce window = %d, want 1", got)
	}
	clk.Advance(time.Second)
	if _, _ = r.Refresh(context.Background(), false); lister.callCount() != 2 {
		t.Fatalf("ListPorts calls after the debounce window = %d, want 2", lister.callCount())
	}
}

func TestPortSetRefresherKeepsSetOnAPIFailure(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080})
	r, clk := newTestRefresher(t, lister, nil, newPortNotifier(), nil)
	if _, fresh := r.Refresh(context.Background(), true); !fresh {
		t.Fatal("first refresh must report a fresh API result")
	}

	lister.mu.Lock()
	lister.err = errors.New("503 service unavailable")
	lister.mu.Unlock()

	clk.Advance(2 * time.Second)
	ports, fresh := r.Refresh(context.Background(), false)
	if fresh {
		t.Fatal("a failed API call must not report a fresh result")
	}
	if len(ports) != 1 || ports[0] != 8080 {
		t.Fatalf("failed refresh must keep the last known port set, got %v", ports)
	}
}

func TestPortSetRefresherRespectsTimeout(t *testing.T) {
	lister := &fakePortLister{block: 5 * time.Second, cancel: true}
	r, _ := newTestRefresher(t, lister, nil, newPortNotifier(), nil)
	r.timeout = 50 * time.Millisecond

	start := time.Now()
	ports, fresh := r.Refresh(context.Background(), true)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("refresh took %s, must be bounded by the refresh timeout", elapsed)
	}
	if fresh {
		t.Fatal("a timed-out refresh must not report a fresh result")
	}
	if len(ports) != 0 {
		t.Fatalf("ports = %v, want empty", ports)
	}
}

func TestPortSetRefresherCancelledHostContext(t *testing.T) {
	lister := &fakePortLister{cancel: true}
	lister.set([]int{8080})
	r, _ := newTestRefresher(t, lister, nil, newPortNotifier(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ports, fresh := r.Refresh(ctx, true)
	if fresh {
		t.Fatal("a cancelled host context must not report a fresh result")
	}
	if len(ports) != 0 {
		t.Fatalf("ports = %v, want empty", ports)
	}
}

func TestPortSetRefresherWithoutAPICredential(t *testing.T) {
	lister := &fakePortLister{}
	lister.set([]int{8080, 18081})
	pn := newPortNotifier()
	pn.update([]int{8080})
	r := newPortSetRefresher(&Devbridge{logger: slog.Default(), outputWriter: &bytes.Buffer{}}, "abcd2345", nil, pn, nil)
	r.listPorts = lister.list

	ports, fresh := r.Refresh(context.Background(), true)
	if fresh {
		t.Fatal("--token mode (no REST credential) must not report a fresh API result")
	}
	if len(ports) != 1 || ports[0] != 8080 {
		t.Fatalf("ports = %v, want the gateway-delivered [8080]", ports)
	}
	if got := lister.callCount(); got != 0 {
		t.Fatalf("--token mode must not call ListPorts, calls = %d", got)
	}
}

func TestPortNotifierConcurrentUpdates(t *testing.T) {
	pn := newPortNotifier()
	var wg sync.WaitGroup
	var updates int64
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ports := []int{8080, 3000 + worker}
				pn.update(ports)
				atomic.AddInt64(&updates, 1)
				_ = pn.currentPorts()
			}
		}(i)
	}
	wg.Wait()
	if got := atomic.LoadInt64(&updates); got != 400 {
		t.Fatalf("updates = %d, want 400", got)
	}
	if got := pn.currentPorts(); len(got) != 2 {
		t.Fatalf("currentPorts = %v, want 2 ports", got)
	}
}

func TestDiffPorts(t *testing.T) {
	added, removed := diffPorts([]int{80, 443}, []int{443, 8080})
	if len(added) != 1 || added[0] != 8080 {
		t.Fatalf("added = %v, want [8080]", added)
	}
	if len(removed) != 1 || removed[0] != 80 {
		t.Fatalf("removed = %v, want [80]", removed)
	}
}

func TestDiffPortsIdempotent(t *testing.T) {
	added, removed := diffPorts([]int{8080, 18081}, []int{8080, 18081})
	if len(added) != 0 || len(removed) != 0 {
		t.Fatalf("identical sets must diff empty, got added=%v removed=%v", added, removed)
	}
}

func TestFormatPortList(t *testing.T) {
	if got := formatPortList(nil); got != "-" {
		t.Fatalf("formatPortList(nil) = %q, want %q", got, "-")
	}
	if got := formatPortList([]int{3000, 8080, 80}); got != "80,3000,8080" {
		t.Fatalf("formatPortList must sort ports, got %q", got)
	}
}

func TestPortNumbers(t *testing.T) {
	got := portNumbers([]Port{{Port: 18081}, {Port: 8080}})
	if len(got) != 2 || got[0] != 18081 || got[1] != 8080 {
		t.Fatalf("portNumbers = %v, want [18081 8080]", got)
	}
	if got := portNumbers(nil); len(got) != 0 {
		t.Fatalf("portNumbers(nil) = %v, want empty", got)
	}
}

func TestTunnelStatusHostOnline(t *testing.T) {
	cases := []struct {
		name string
		s    *TunnelStatus
		want bool
	}{
		{"nil status", nil, false},
		{"zero status", &TunnelStatus{}, false},
		{"host connected", &TunnelStatus{HostConnectionCount: 1}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.HostOnline(); got != tc.want {
				t.Fatalf("HostOnline() = %v, want %v", got, tc.want)
			}
		})
	}
}
