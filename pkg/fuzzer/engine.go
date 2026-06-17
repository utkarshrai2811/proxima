package fuzzer

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when an attack ID is unknown.
var ErrNotFound = errors.New("fuzzer: attack not found")

const maxResponseBody = 1 << 20 // 1 MiB cap on stored response bodies

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}

// job is a single generated request: an index and the per-position values.
type job struct {
	Index  int
	Values map[string]string
}

// generateJobs expands an attack into the full sequence of requests.
func generateJobs(a *Attack) ([]job, error) {
	names, err := PositionNames(a.BaseRequest)
	if err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return nil, errors.New("fuzzer: base request has no §positions§")
	}

	lists := make([][]string, len(a.PayloadSources))
	for i, src := range a.PayloadSources {
		lists[i] = src.Resolve()
	}

	switch a.Type {
	case AttackTypeSniper:
		return sniperJobs(names, firstList(lists)), nil
	case AttackTypeBatteringRam:
		return batteringRamJobs(names, firstList(lists)), nil
	case AttackTypePitchfork:
		return pitchforkJobs(names, lists), nil
	case AttackTypeClusterBomb:
		return clusterBombJobs(names, lists), nil
	default:
		return nil, fmt.Errorf("fuzzer: unknown attack type %q", a.Type)
	}
}

func firstList(lists [][]string) []string {
	if len(lists) > 0 {
		return lists[0]
	}

	return nil
}

func sniperJobs(names, list []string) []job {
	var jobs []job

	for _, pos := range names {
		for _, payload := range list {
			vals := make(map[string]string, len(names))
			for _, n := range names {
				vals[n] = ""
			}

			vals[pos] = payload
			jobs = append(jobs, job{Index: len(jobs), Values: vals})
		}
	}

	return jobs
}

func batteringRamJobs(names, list []string) []job {
	var jobs []job

	for _, payload := range list {
		vals := make(map[string]string, len(names))
		for _, n := range names {
			vals[n] = payload
		}

		jobs = append(jobs, job{Index: len(jobs), Values: vals})
	}

	return jobs
}

func pitchforkJobs(names []string, lists [][]string) []job {
	minLen := -1

	for j := range names {
		if j >= len(lists) {
			break
		}

		if minLen == -1 || len(lists[j]) < minLen {
			minLen = len(lists[j])
		}
	}

	if minLen < 0 {
		minLen = 0
	}

	var jobs []job

	for i := 0; i < minLen; i++ {
		vals := make(map[string]string, len(names))

		for j, pos := range names {
			if j < len(lists) && i < len(lists[j]) {
				vals[pos] = lists[j][i]
			} else {
				vals[pos] = ""
			}
		}

		jobs = append(jobs, job{Index: len(jobs), Values: vals})
	}

	return jobs
}

func clusterBombJobs(names []string, lists [][]string) []job {
	perPos := make([][]string, len(names))

	for j := range names {
		if j < len(lists) && len(lists[j]) > 0 {
			perPos[j] = lists[j]
		} else {
			perPos[j] = []string{""}
		}
	}

	var jobs []job

	idx := make([]int, len(names))
	for {
		vals := make(map[string]string, len(names))
		for j, pos := range names {
			vals[pos] = perPos[j][idx[j]]
		}

		jobs = append(jobs, job{Index: len(jobs), Values: vals})

		k := len(names) - 1
		for k >= 0 {
			idx[k]++
			if idx[k] < len(perPos[k]) {
				break
			}

			idx[k] = 0
			k--
		}

		if k < 0 {
			break
		}
	}

	return jobs
}

// control coordinates pause/resume/cancel for a running attack.
type control struct {
	mu        sync.Mutex
	cond      *sync.Cond
	paused    bool
	cancelled bool
}

func newControl() *control {
	c := &control{}
	c.cond = sync.NewCond(&c.mu)

	return c
}

func (c *control) waitIfPaused() (cancelled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for c.paused && !c.cancelled {
		c.cond.Wait()
	}

	return c.cancelled
}

func (c *control) pause()  { c.set(func() { c.paused = true }) }
func (c *control) resume() { c.set(func() { c.paused = false }) }
func (c *control) cancel() { c.set(func() { c.cancelled = true }) }

func (c *control) set(fn func()) {
	c.mu.Lock()
	fn()
	c.mu.Unlock()
	c.cond.Broadcast()
}

func (c *control) isCancelled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cancelled
}

// Manager owns attacks, their results, and live result subscribers (in memory).
type Manager struct {
	mu       sync.RWMutex
	attacks  map[string]*Attack
	order    []string
	results  map[string][]FuzzResult
	subs     map[string]map[chan FuzzResult]struct{}
	controls map[string]*control
	client   *http.Client
	onResult func(FuzzResult) // optional hook (wired by the plugin system in 4D)
}

func NewManager() *Manager {
	return &Manager{
		attacks:  make(map[string]*Attack),
		results:  make(map[string][]FuzzResult),
		subs:     make(map[string]map[chan FuzzResult]struct{}),
		controls: make(map[string]*control),
		client: &http.Client{
			Timeout: 30 * time.Second,
			// A fuzzer commonly targets test hosts with self-signed certs, so
			// certificate errors are ignored (matching Burp Intruder behaviour).
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
	}
}

// SetResultHook registers a callback invoked for every completed result.
func (m *Manager) SetResultHook(fn func(FuzzResult)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onResult = fn
}

// AttackInput is the request to create an attack.
type AttackInput struct {
	Name           string
	Type           AttackType
	BaseRequest    string
	PayloadSources []PayloadSource
	Concurrency    int
}

func (m *Manager) CreateAttack(in AttackInput) (*Attack, error) {
	concurrency := in.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	if concurrency > 50 {
		concurrency = 50
	}

	a := &Attack{
		ID:             newID(),
		Name:           in.Name,
		Type:           in.Type,
		BaseRequest:    in.BaseRequest,
		PayloadSources: in.PayloadSources,
		Concurrency:    concurrency,
		Status:         AttackStatusPending,
		CreatedAt:      time.Now(),
	}

	jobs, err := generateJobs(a)
	if err != nil {
		return nil, err
	}

	a.TotalRequests = len(jobs)

	m.mu.Lock()
	m.attacks[a.ID] = a
	m.order = append(m.order, a.ID)
	m.mu.Unlock()

	return a, nil
}

func (m *Manager) GetAttack(id string) (Attack, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if a, ok := m.attacks[id]; ok {
		return *a, true
	}

	return Attack{}, false
}

func (m *Manager) ListAttacks() []Attack {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Attack, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, *m.attacks[id])
	}

	return out
}

func (m *Manager) ListResults(attackID string) []FuzzResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]FuzzResult(nil), m.results[attackID]...)
}

func (m *Manager) Subscribe(attackID string) (<-chan FuzzResult, func()) {
	ch := make(chan FuzzResult, 64)

	m.mu.Lock()
	if m.subs[attackID] == nil {
		m.subs[attackID] = make(map[chan FuzzResult]struct{})
	}
	m.subs[attackID][ch] = struct{}{}
	m.mu.Unlock()

	return ch, func() {
		m.mu.Lock()
		delete(m.subs[attackID], ch)
		m.mu.Unlock()
	}
}

func (m *Manager) Start(id string) error {
	m.mu.Lock()

	a, ok := m.attacks[id]
	if !ok {
		m.mu.Unlock()

		return ErrNotFound
	}

	switch a.Status {
	case AttackStatusPending:
		a.Status = AttackStatusRunning
		now := time.Now()
		a.StartedAt = &now
		ctrl := newControl()
		m.controls[id] = ctrl
		m.mu.Unlock()

		go m.run(a, ctrl)

		return nil
	case AttackStatusPaused:
		a.Status = AttackStatusRunning
		ctrl := m.controls[id]
		m.mu.Unlock()

		if ctrl != nil {
			ctrl.resume()
		}

		return nil
	default:
		status := a.Status
		m.mu.Unlock()

		return fmt.Errorf("fuzzer: attack is %s, cannot start", status)
	}
}

func (m *Manager) Pause(id string) error {
	m.mu.Lock()

	a, ok := m.attacks[id]
	if !ok {
		m.mu.Unlock()

		return ErrNotFound
	}

	if a.Status != AttackStatusRunning {
		m.mu.Unlock()

		return fmt.Errorf("fuzzer: attack is %s, cannot pause", a.Status)
	}

	a.Status = AttackStatusPaused
	ctrl := m.controls[id]
	m.mu.Unlock()

	if ctrl != nil {
		ctrl.pause()
	}

	return nil
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()

	a, ok := m.attacks[id]
	if !ok {
		m.mu.Unlock()

		return ErrNotFound
	}

	if a.Status == AttackStatusDone || a.Status == AttackStatusCancelled {
		m.mu.Unlock()

		return nil
	}

	ctrl := m.controls[id]

	if ctrl == nil {
		// Pending and never started: mark cancelled directly.
		a.Status = AttackStatusCancelled
		now := time.Now()
		a.FinishedAt = &now
		m.mu.Unlock()

		return nil
	}

	m.mu.Unlock()
	ctrl.cancel()

	return nil
}

func (m *Manager) run(a *Attack, ctrl *control) {
	jobs, err := generateJobs(a)
	if err != nil {
		return
	}

	jobCh := make(chan job)

	var wg sync.WaitGroup

	for i := 0; i < a.Concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for jb := range jobCh {
				if ctrl.isCancelled() {
					continue
				}

				m.addResult(a.ID, m.execute(a, jb))
			}
		}()
	}

	for i := 0; i < len(jobs); i++ {
		if ctrl.waitIfPaused() {
			break
		}

		jobCh <- jobs[i]
	}

	close(jobCh)
	wg.Wait()
	m.finish(a.ID, ctrl)
}

func (m *Manager) finish(id string, ctrl *control) {
	m.mu.Lock()
	defer m.mu.Unlock()

	a, ok := m.attacks[id]
	if !ok {
		return
	}

	now := time.Now()
	a.FinishedAt = &now

	if ctrl.isCancelled() {
		a.Status = AttackStatusCancelled
	} else {
		a.Status = AttackStatusDone
	}

	delete(m.controls, id)
}

func (m *Manager) addResult(attackID string, res FuzzResult) {
	m.mu.Lock()

	m.results[attackID] = append(m.results[attackID], res)

	if a, ok := m.attacks[attackID]; ok {
		a.CompletedCount++
		if res.IsError {
			a.ErrorCount++
		}
	}

	chans := make([]chan FuzzResult, 0, len(m.subs[attackID]))
	for ch := range m.subs[attackID] {
		chans = append(chans, ch)
	}

	hook := m.onResult
	m.mu.Unlock()

	for _, ch := range chans {
		select {
		case ch <- res:
		default:
		}
	}

	if hook != nil {
		hook(res)
	}
}

func (m *Manager) execute(a *Attack, jb job) FuzzResult {
	raw := Substitute(a.BaseRequest, jb.Values)
	res := FuzzResult{
		ID: newID(), AttackID: a.ID, RequestIndex: jb.Index,
		PayloadValues: jb.Values, RawRequest: raw,
	}

	req, err := parseRawRequest(raw)
	if err != nil {
		res.IsError = true
		res.ErrorMessage = err.Error()

		return res
	}

	start := time.Now()

	resp, err := m.client.Do(req)
	res.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		res.IsError = true
		res.ErrorMessage = err.Error()

		return res
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	res.StatusCode = resp.StatusCode
	res.ResponseSize = len(body)
	res.RawResponse = formatResponse(resp, body)

	return res
}

func parseRawRequest(raw string) (*http.Request, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	head := raw
	body := ""

	if sep := strings.Index(raw, "\n\n"); sep != -1 {
		head = raw[:sep]
		body = raw[sep+2:]
	}

	lines := strings.Split(head, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil, errors.New("fuzzer: empty request")
	}

	parts := strings.Fields(lines[0])
	if len(parts) < 2 {
		return nil, errors.New("fuzzer: request line must be 'METHOD URL [PROTO]'")
	}

	req, err := http.NewRequest(parts[0], parts[1], strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("fuzzer: invalid request: %w", err)
	}

	if req.URL.Scheme == "" || req.URL.Host == "" {
		return nil, errors.New("fuzzer: request line must use a full URL (scheme://host/path)")
	}

	for _, line := range lines[1:] {
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}

		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])

		switch {
		case strings.EqualFold(k, "Host"):
			req.Host = v
		case strings.EqualFold(k, "Content-Length"):
		default:
			req.Header.Add(k, v)
		}
	}

	return req, nil
}

func formatResponse(resp *http.Response, body []byte) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s %s\n", resp.Proto, resp.Status)

	for k, vv := range resp.Header {
		for _, v := range vv {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}

	b.WriteString("\n")
	b.Write(body)

	return b.String()
}
