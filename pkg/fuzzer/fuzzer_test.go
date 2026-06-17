package fuzzer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParsePositionsAndSubstitute(t *testing.T) {
	t.Parallel()

	tmpl := "GET http://x/?a=§p1§&b=§p2§ HTTP/1.1\n\n"

	names, err := PositionNames(tmpl)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 2 || names[0] != "p1" || names[1] != "p2" {
		t.Fatalf("names = %v", names)
	}

	got := Substitute(tmpl, map[string]string{"p1": "A", "p2": "B"})
	if got != "GET http://x/?a=A&b=B HTTP/1.1\n\n" {
		t.Fatalf("substitute = %q", got)
	}

	if _, err := ParsePositions("oops §unclosed"); err == nil {
		t.Error("expected error for unclosed marker")
	}
}

func TestBuiltInPayloads(t *testing.T) {
	t.Parallel()

	if got := BuiltInPayloads(BuiltInNumericRange, 1, 5); len(got) != 5 || got[0] != "1" || got[4] != "5" {
		t.Fatalf("numeric range = %v", got)
	}

	if got := BuiltInPayloads(BuiltInSQLiBasic, 0, 0); len(got) < 10 {
		t.Fatalf("expected a populated sqli list, got %d", len(got))
	}
}

func names2() []string { return []string{"p1", "p2"} }

func TestGenerateJobs(t *testing.T) {
	t.Parallel()

	tmpl := "§p1§ §p2§"
	mk := func(typ AttackType, sources []PayloadSource) []job {
		a := &Attack{Type: typ, BaseRequest: tmpl, PayloadSources: sources}
		jobs, err := generateJobs(a)
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}

		return jobs
	}

	inline := func(v ...string) PayloadSource {
		return PayloadSource{Type: PayloadSourceInline, Values: v}
	}

	// Sniper: 2 positions × 2 payloads = 4, each sets exactly one position.
	sniper := mk(AttackTypeSniper, []PayloadSource{inline("A", "B")})
	if len(sniper) != 4 {
		t.Fatalf("sniper jobs = %d, want 4", len(sniper))
	}

	if sniper[0].Values["p1"] != "A" || sniper[0].Values["p2"] != "" {
		t.Errorf("sniper[0] = %v", sniper[0].Values)
	}

	// Battering ram: 2 payloads, both positions share the value.
	ram := mk(AttackTypeBatteringRam, []PayloadSource{inline("A", "B")})
	if len(ram) != 2 || ram[0].Values["p1"] != "A" || ram[0].Values["p2"] != "A" {
		t.Fatalf("battering ram = %v", ram)
	}

	// Pitchfork: lockstep, stops at shortest (2).
	pitch := mk(AttackTypePitchfork, []PayloadSource{inline("A", "B"), inline("X", "Y", "Z")})
	if len(pitch) != 2 || pitch[1].Values["p1"] != "B" || pitch[1].Values["p2"] != "Y" {
		t.Fatalf("pitchfork = %v", pitch)
	}

	// Cluster bomb: cartesian product = 2 × 2 = 4.
	cluster := mk(AttackTypeClusterBomb, []PayloadSource{inline("A", "B"), inline("X", "Y")})
	if len(cluster) != 4 {
		t.Fatalf("cluster bomb jobs = %d, want 4", len(cluster))
	}

	_ = names2
}

func TestEngineRun(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "echo %s", r.URL.Query().Get("q"))
	}))
	defer srv.Close()

	m := NewManager()
	attack, err := m.CreateAttack(AttackInput{
		Name:           "test",
		Type:           AttackTypeSniper,
		BaseRequest:    "GET " + srv.URL + "/?q=§p§ HTTP/1.1\n\n",
		PayloadSources: []PayloadSource{{Type: PayloadSourceInline, Values: []string{"a", "b", "c"}}},
		Concurrency:    2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if attack.TotalRequests != 3 {
		t.Fatalf("TotalRequests = %d, want 3", attack.TotalRequests)
	}

	if err := m.Start(attack.ID); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(5 * time.Second)
	for {
		a, _ := m.GetAttack(attack.ID)
		if a.Status == AttackStatusDone {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("attack did not finish; status=%s completed=%d", a.Status, a.CompletedCount)
		case <-time.After(20 * time.Millisecond):
		}
	}

	results := m.ListResults(attack.ID)
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}

	for _, res := range results {
		if res.IsError || res.StatusCode != http.StatusOK {
			t.Errorf("result error/status: %+v", res)
		}
	}
}

func TestCancelPending(t *testing.T) {
	t.Parallel()

	m := NewManager()
	attack, err := m.CreateAttack(AttackInput{
		Name:           "c",
		Type:           AttackTypeSniper,
		BaseRequest:    "GET http://127.0.0.1:1/?q=§p§ HTTP/1.1\n\n",
		PayloadSources: []PayloadSource{{Type: PayloadSourceInline, Values: []string{"a"}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Cancel(attack.ID); err != nil {
		t.Fatal(err)
	}

	if a, _ := m.GetAttack(attack.ID); a.Status != AttackStatusCancelled {
		t.Fatalf("status = %s, want CANCELLED", a.Status)
	}
}
